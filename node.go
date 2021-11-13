package jtree

import (
	"encoding"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type globalOptions struct {
	noUnknown bool
	typeReg   *TypeRegistry
	encReg    *EncodingRegistry
}

func (g *globalOptions) types() *TypeRegistry {
	if g.typeReg != nil {
		return g.typeReg
	}
	return defaultTypeRegistry
}

func (g *globalOptions) encodings() *EncodingRegistry {
	if g.encReg != nil {
		return g.encReg
	}
	return defaultEncodingRegistry
}

type decoderOptions struct {
	str    bool
	enc    Encoding
	global *globalOptions
}

func (o *decoderOptions) apply(opts []Option) *decoderOptions {
	for _, fn := range opts {
		fn(o)
	}
	return o
}

func (o *decoderOptions) g() *globalOptions {
	if o.global != nil {
		return o.global
	}
	o.global = new(globalOptions)
	return o.global
}

// OpString makes the numeric value to be encoded/decoded as a string and the byte slice value
// to be converted to a string as is (skips the binary encoding scheme)
func OpString(o *decoderOptions) { o.str = true }

// OpEncoding specifies the binary encoding scheme used for byte slices. Without this option base64 scheme will be used
func OpEncoding(e Encoding) Option { return func(o *decoderOptions) { o.enc = e } }

// OpTypes provides custom user type registry. The option is global for all Decode calls in chain
func OpTypes(r *TypeRegistry) Option { return func(o *decoderOptions) { o.g().typeReg = r } }

// OpEncodings provides custom user encodings registry. The option is global for all Decode calls in chain
func OpEncodings(e *EncodingRegistry) Option { return func(o *decoderOptions) { o.g().encReg = e } }

// OpDisallowUnknownFields causes the Decode method to return an error when the destination is a struct
// and the input contains object keys which do not match any non-ignored, exported fields in the destination.
func OpDisallowUnknownFields(o *decoderOptions) { o.g().noUnknown = true }

// OpGlobal passes global options to subsequent Decode calls. Used in custom decoders
func OpGlobal(op []Option) Option {
	return func(o *decoderOptions) {
		src := new(decoderOptions).apply(op)
		o.global = src.global
	}
}

func opG(src *decoderOptions) Option { return func(o *decoderOptions) { o.global = src.global } }

// Option is a function pointer used to pass options to Decode method
type Option func(*decoderOptions)

// Node is a JSON AST node
type Node interface {
	// Type returns Node type name: "number", "string", "object", "array", "boolean" or "null"
	Type() string
	// Decode decodes the node into the value pointed by v
	Decode(v interface{}, op ...Option) error
	/*
		// TODO
		String() string
		WriteTo(w io.Writer) (int64, error)
	*/
}

// Num represents numeric node
type Num big.Float // on conversion operations the difference in performance between big.Float and big.Int is insignificant

// Type returns the node type i.e. "number"
func (*Num) Type() string { return "number" }

// Decode decodes the node into the value pointed by v
func (n *Num) Decode(v interface{}, op ...Option) error {
	fn := func(out reflect.Value) error {
		switch out.Type() {
		case bigIntType:
			i, _ := (*big.Float)(n).Int(nil)
			out.Set(reflect.ValueOf(*i))

		case bigFloatType:
			out.Set(reflect.ValueOf(*(*big.Float)(n)))

		case timeType:
			u, _ := (*big.Float)(n).Int64()
			tmp := time.Unix(u, 0).UTC()
			out.Set(reflect.ValueOf(tmp))

		default:
			k := out.Kind()
			switch {
			case k >= reflect.Int && k <= reflect.Int64:
				i, _ := (*big.Float)(n).Int64()
				out.SetInt(i)

			case k >= reflect.Uint && k <= reflect.Uintptr:
				u, _ := (*big.Float)(n).Uint64()
				out.SetUint(u)

			case k == reflect.Float32 || k == reflect.Float64:
				f, _ := (*big.Float)(n).Float64()
				out.SetFloat(f)

			case k == reflect.String:
				out.SetString((*big.Float)(n).String())

			case k == reflect.Bool:
				v := (*big.Float)(n).Cmp(big.NewFloat(0)) != 0
				out.SetBool(v)

			default:
				return fmt.Errorf("jtree: can't convert number to %v", out.Type())
			}
		}
		return nil
	}
	return decodeNode(v, n, fn, op...)
}

// String represents string node
type String string

// Type returns the node i.e. "string"
func (String) Type() string { return "string" }

// Decode decodes the node into the value pointed by v
func (s String) Decode(v interface{}, op ...Option) error {
	opt := new(decoderOptions).apply(op)
	fn := func(out reflect.Value) error {
		t := out.Type()
		switch {
		case reflect.PtrTo(t).Implements(textUnmarshalerType) && out.CanAddr():
			unmarshaler := out.Addr().Interface().(encoding.TextUnmarshaler)
			if err := unmarshaler.UnmarshalText([]byte(s)); err != nil {
				return fmt.Errorf("jtree: %w", err)
			}

		case t.Kind() == reflect.String || t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8:
			var src reflect.Value
			enc := opt.enc
			if enc == nil && t.Kind() != reflect.String && !opt.str {
				enc = Base64
			}
			if enc != nil {
				buf, err := enc.Decode([]byte(s))
				if err != nil {
					return fmt.Errorf("jtree: %w", err)
				}
				src = reflect.ValueOf(buf)
			} else {
				src = reflect.ValueOf(string(s))
			}
			if !src.CanConvert(t) {
				return fmt.Errorf("jtree: can't convert string to %v", t)
			}
			out.Set(src.Convert(t))

		default:
			if !opt.str {
				return fmt.Errorf("jtree: can't convert string to %v", t)
			}
			k := out.Kind()
			switch {
			case t == bigIntType:
				i, ok := new(big.Int).SetString(string(s), 10)
				if !ok {
					return fmt.Errorf("jtree: error parsing integer number: %s", s)
				}
				out.Set(reflect.ValueOf(*i))

			case t == bigFloatType:
				f, _, err := new(big.Float).Parse(string(s), 10)
				if err != nil {
					return fmt.Errorf("jtree: %w", err)
				}
				out.Set(reflect.ValueOf(*f))

			case k >= reflect.Int && k <= reflect.Int64:
				i, err := strconv.ParseInt(string(s), 10, 64)
				if err != nil {
					return fmt.Errorf("jtree: %w", err)
				}
				out.SetInt(i)

			case k >= reflect.Uint && k <= reflect.Uintptr:
				i, err := strconv.ParseUint(string(s), 10, 64)
				if err != nil {
					return fmt.Errorf("jtree: %w", err)
				}
				out.SetUint(i)

			case k == reflect.Bool:
				v, err := strconv.ParseBool(string(s))
				if err != nil {
					return fmt.Errorf("jtree: %w", err)
				}
				out.SetBool(v)

			default:
				return fmt.Errorf("jtree: can't convert string to %v", t)
			}
		}
		return nil
	}
	return decodeNode(v, s, fn, op...)
}

// Object represents object node
type Object struct {
	keys   []string
	values map[string]Node
}

// Type returns the node i.e. "object"
func (*Object) Type() string { return "object" }

// Field is used to construct objects
type Field struct {
	Key   string
	Value Node
}

// Fields is used to construct objects
type Fields []*Field

// NewObject returns new Object node
func (f Fields) NewObject() *Object {
	object := Object{
		keys:   make([]string, len(f)),
		values: make(map[string]Node, len(f)),
	}
	for i, n := range f {
		object.keys[i] = n.Key
		object.values[n.Key] = n.Value
	}
	return &object
}

// Keys returns all object keys
func (o *Object) Keys() []string {
	return o.keys
}

// FieldByName returns the field with specific name or nil
func (o *Object) FieldByName(f string) Node {
	return o.values[f]
}

// Field returns i'th field or nil if the number of fields is exceeded
func (o *Object) Field(i int) (string, Node) {
	if i >= len(o.keys) {
		return "", nil
	}
	return o.keys[i], o.values[o.keys[i]]
}

// NumField returns the number of fields
func (o *Object) NumField() int {
	return len(o.keys)
}

// Decode decodes the node into the value pointed by v
func (o *Object) Decode(v interface{}, op ...Option) error {
	opt := new(decoderOptions).apply(op)
	fn := func(out reflect.Value) error {
		t := out.Type()
		switch t.Kind() {
		case reflect.Struct:
			fields := make(map[string]*structField)
			collectFields(t, nil, nil, fields)
			for i := 0; i < o.NumField(); i++ {
				key, elem := o.Field(i)
				field, ok := fields[key]
				if !ok {
					if opt.g().noUnknown {
						return fmt.Errorf("jtree: undefined field '%s': %v", key, out.Type())
					}
					continue
				}
				dest := out
				for i, fi := range field.Index {
					dest = dest.Field(fi)
					if i < len(field.Index)-1 && dest.Kind() == reflect.Ptr {
						// allocate anonymous fields
						if dest.IsNil() {
							dest.Set(reflect.New(dest.Type().Elem()))
						}
						dest = dest.Elem()
					}
				}
				fopt := parseFieldOptions(field.opt, opt)
				if err := elem.Decode(dest.Addr().Interface(), append([]Option{opG(opt)}, fopt...)...); err != nil {
					return err
				}
			}
			return nil

		case reflect.Map:
			dst := reflect.MakeMap(t)
			for i := 0; i < o.NumField(); i++ {
				key, elem := o.Field(i)
				keyVal := reflect.New(t.Key())
				if err := String(key).Decode(keyVal.Interface(), OpString); err != nil {
					return err
				}
				elemVal := reflect.New(t.Elem())
				if err := elem.Decode(elemVal.Interface(), opG(opt)); err != nil {
					return err
				}
				dst.SetMapIndex(keyVal.Elem(), elemVal.Elem())
			}
			out.Set(dst)
			return nil

		default:
			return fmt.Errorf("jtree: struct or map expected: %v", t)
		}
	}
	return decodeNode(v, o, fn, op...)
}

// Array represents JSON array
type Array []Node

// Type returns the node i.e. "array"
func (Array) Type() string { return "array" }

// Decode decodes the node into the value pointed by v
func (a Array) Decode(v interface{}, op ...Option) error {
	opt := new(decoderOptions).apply(op)
	fn := func(out reflect.Value) error {
		var dst reflect.Value
		switch out.Kind() {
		case reflect.Slice:
			dst = reflect.MakeSlice(out.Type(), len(a), len(a))
		case reflect.Array:
			dst = out
		default:
			return fmt.Errorf("jtree: slice or array expected: %v", out.Type())
		}
		for i, elem := range a {
			if i == dst.Len() {
				break
			}
			if err := elem.Decode(dst.Index(i).Addr().Interface(), opG(opt)); err != nil {
				return err
			}
		}
		if dst != out {
			out.Set(dst)
		}
		return nil
	}
	return decodeNode(v, a, fn, op...)
}

// Array represents boolean node
type Bool bool

// Type returns the node i.e. "boolean"
func (Bool) Type() string { return "boolean" }

// Decode decodes the node into the value pointed by v
func (b Bool) Decode(v interface{}, op ...Option) error {
	fn := func(out reflect.Value) error {
		k := out.Kind()
		switch k {
		case reflect.Bool:
			out.SetBool(bool(b))

		case reflect.String:
			out.SetString(strconv.FormatBool(bool(b)))

		default:
			v := 0
			if b {
				v = 1
			}
			src := reflect.ValueOf(v)
			if !src.CanConvert(out.Type()) {
				return fmt.Errorf("jtree: can't convert boolean to %v", out.Type())
			}
			out.Set(src.Convert(out.Type()))
		}
		return nil
	}
	return decodeNode(v, b, fn, op...)
}

// Array represents null node
type Null struct{}

// Type returns the node i.e. "null"
func (Null) Type() string { return "null" }

// Decode decodes the node into the value pointed by v
func (n Null) Decode(v interface{}, op ...Option) error {
	return decodeNode(v, n, nil, op...)
}

var (
	nodeType            = reflect.TypeOf((*Node)(nil)).Elem()
	textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
	bigIntType          = reflect.TypeOf((*big.Int)(nil)).Elem()
	bigFloatType        = reflect.TypeOf((*big.Float)(nil)).Elem()
	timeType            = reflect.TypeOf((*time.Time)(nil)).Elem()
	emptyType           = reflect.TypeOf((*interface{})(nil)).Elem()
	errorType           = reflect.TypeOf((*error)(nil)).Elem()
	float64Type         = reflect.TypeOf(float64(0))
	stringType          = reflect.TypeOf("")
	boolType            = reflect.TypeOf(false)
	objectType          = reflect.MapOf(stringType, emptyType)
	arrayType           = reflect.SliceOf(emptyType)
	optionsType         = reflect.SliceOf(reflect.TypeOf((*Option)(nil)).Elem())
)

type decodeFunc func(out reflect.Value) error

func decodeNode(v interface{}, node Node, decode decodeFunc, op ...Option) error {
	opt := new(decoderOptions).apply(op)
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("jtree: pointer expected: %v", val.Type())
	}
	if val.IsNil() {
		return errors.New("jtree: nil pointer")
	}
	out := val.Elem()
	if _, ok := node.(Null); ok {
		// special case
		out.Set(reflect.Zero(out.Type()))
		return nil
	}

	// out must be a non pointer
	for out.Kind() == reflect.Ptr {
		if out.IsNil() {
			out.Set(reflect.New(out.Type().Elem()))
		}
		out = out.Elem()
	}

	if out.Kind() != reflect.Interface {
		if err := decode(out); err != nil {
			return err
		}
	} else {
		if out.Type() == nodeType {
			// special case
			out.Set(reflect.ValueOf(node))
		} else {
			v, err := opt.g().types().call(out.Type(), node, op)
			if err != nil {
				return fmt.Errorf("jtree: %w", err)
			}
			if v.IsValid() {
				out.Set(v)
			} else {
				// allocate default type
				var dst reflect.Value
				switch node.(type) {
				case *Num:
					dst = reflect.New(float64Type).Elem()
				case String:
					dst = reflect.New(stringType).Elem()
				case *Object:
					dst = reflect.New(objectType).Elem()
				case Array:
					dst = reflect.New(arrayType).Elem()
				case Bool:
					dst = reflect.New(boolType).Elem()
				default:
					panic("unknown node")
				}
				if err := decode(dst); err != nil {
					return err
				}
				if !dst.CanConvert(out.Type()) {
					return fmt.Errorf("jtree: can't convert %v to %v", dst.Type(), out.Type())
				}
				out.Set(dst.Convert(out.Type()))
			}
		}
	}
	return nil
}

func parseTag(tag string) (name string, opt []string) {
	s := strings.Split(tag, ",")
	return s[0], s[1:]
}

func parseFieldOptions(tags []string, opt *decoderOptions) []Option {
	o := make([]Option, 0)
	for _, s := range tags {
		if s == "string" {
			o = append(o, OpString)
		} else if enc := opt.g().encodings().get(s); enc != nil {
			o = append(o, OpEncoding(enc))
		}
	}
	return o
}

type structField struct {
	*reflect.StructField
	opt []string
}

func mkIndex(a, b []int) []int {
	cp := make([]int, len(a)+len(b))
	copy(cp, a)
	copy(cp[len(a):], b)
	return cp
}

func collectFields(t reflect.Type, index []int, ptr []reflect.Type, out map[string]*structField) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name, opt := parseTag(string(f.Tag.Get("json")))
		if name == "-" {
			continue
		}
		if name == "" && f.Anonymous && (f.Type.Kind() == reflect.Struct || f.Type.Kind() == reflect.Ptr && f.Type.Elem().Kind() == reflect.Struct) {
			// dive
			ft := f.Type
			if ft.Kind() == reflect.Ptr {
				if !f.IsExported() {
					continue
				}
				i := 0
				for ; i < len(ptr) && ptr[i] != ft; i++ {
				}
				if i < len(ptr) {
					// loop detected
					continue
				}
				ptr = append(ptr, ft)
				ft = f.Type.Elem()
			}
			collectFields(ft, mkIndex(index, f.Index), ptr, out)
		} else if !f.IsExported() {
			continue
		} else {
			if name == "" {
				name = f.Name
			}
			if prev, ok := out[name]; ok {
				// we use simplified duplicated fields visibility rule here: shallowest and topmost wins
				if len(prev.Index) <= len(f.Index) {
					continue
				}
			}
			tmp := f
			tmp.Index = mkIndex(index, f.Index)
			out[name] = &structField{
				StructField: &tmp,
				opt:         opt,
			}
		}
	}
}
