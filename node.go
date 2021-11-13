package jtree

import (
	"encoding"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Encoding interface {
	Encode([]byte) []byte
	Decode([]byte) ([]byte, error)
}

type base64Encoding struct{}

func (base64Encoding) Encode(src []byte) []byte {
	buf := make([]byte, base64.StdEncoding.EncodedLen(len(src)))
	base64.StdEncoding.Encode(buf, src)
	return buf
}

func (base64Encoding) Decode(src []byte) ([]byte, error) {
	buf := make([]byte, base64.StdEncoding.DecodedLen(len(src)))
	n, err := base64.StdEncoding.Decode(buf, src)
	return buf[:n], err
}

type hexEncoding struct{}

func (hexEncoding) Encode(src []byte) []byte {
	buf := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(buf, src)
	return buf
}

func (hexEncoding) Decode(src []byte) ([]byte, error) {
	buf := make([]byte, hex.DecodedLen(len(src)))
	n, err := hex.Decode(buf, src)
	return buf[:n], err
}

var (
	Base64 Encoding = base64Encoding{}
	Hex    Encoding = hexEncoding{}
)

type globalOptions struct {
	noUnknown bool
	reg       *TypeRegistry
}

func (g *globalOptions) types() *TypeRegistry {
	if g.reg != nil {
		return g.reg
	}
	return defaultRegistry
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

// Node options
func OpString(o *decoderOptions)   { o.str = true }
func OpEncoding(e Encoding) Option { return func(o *decoderOptions) { o.enc = e } }

// Global options
func OpTypes(r *TypeRegistry) Option            { return func(o *decoderOptions) { o.g().reg = r } }
func OpDisallowUnknownFields(o *decoderOptions) { o.g().noUnknown = true }
func OpGlobal(op []Option) Option {
	return func(o *decoderOptions) {
		src := new(decoderOptions).apply(op)
		o.global = src.global
	}
}

func opG(src *decoderOptions) Option { return func(o *decoderOptions) { o.global = src.global } }

type Option func(*decoderOptions)

type Node interface {
	Type() string
	Decode(v interface{}, op ...Option) error
	/*
		// TODO
		String() string
		WriteTo(w io.Writer) (int64, error)
	*/
}

type Num big.Float // on conversion operations the difference in performance between big.Float and big.Int is insignificant

func (*Num) Type() string { return "number" }

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

type String string

func (String) Type() string { return "string" }

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

type Object struct {
	keys   []string
	values map[string]Node
}

func (*Object) Type() string { return "object" }

type Field struct {
	Key   string
	Value Node
}

type Fields []*Field

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

func (o *Object) Keys() []string {
	return o.keys
}

func (o *Object) FieldByName(f string) Node {
	return o.values[f]
}

func (o *Object) Field(i int) (string, Node) {
	if i >= len(o.keys) {
		return "", nil
	}
	return o.keys[i], o.values[o.keys[i]]
}

func (o *Object) NumField() int {
	return len(o.keys)
}

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
				if err := elem.Decode(dest.Addr().Interface(), append([]Option{opG(opt)}, field.opt...)...); err != nil {
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

type Array []Node

func (Array) Type() string { return "array" }

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

type Bool bool

func (Bool) Type() string { return "boolean" }

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

type Null struct{}

func (Null) Type() string { return "null" }

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

func parseTag(tag string) (name string, opt []Option) {
	s := strings.Split(tag, ",")
	name = s[0]
	opt = make([]Option, 0)
	for _, s := range s[1:] {
		if s == "string" {
			opt = append(opt, OpString)
		} else if enc, ok := encoders[s]; ok {
			opt = append(opt, OpEncoding(enc))
		}
	}
	return
}

type structField struct {
	*reflect.StructField
	opt []Option
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

var encoders = map[string]Encoding{
	"base64": Base64,
	"hex":    Hex,
}

func RegisterEncoding(tag string, enc Encoding) {
	encoders[tag] = enc
}

type TypeRegistry struct {
	types map[reflect.Type]interface{}
	mtx   sync.RWMutex
}

func NewTypeRegistry() *TypeRegistry {
	return &TypeRegistry{
		types: make(map[reflect.Type]interface{}),
	}
}

// RegisterType registers new interface type. The argument is a constructor function of type `func(Node, []Option) (UserType, error)`.
// It panics if any other type is passed
func (r *TypeRegistry) RegisterType(fn interface{}) {
	ft := reflect.TypeOf(fn)
	if ft.Kind() != reflect.Func {
		panic(fmt.Sprintf("jtree: function expected: %v", ft))
	}
	if ft.NumIn() != 2 || ft.In(0) != nodeType || ft.In(1) != optionsType || ft.NumOut() != 2 || ft.Out(1) != errorType {
		panic(fmt.Sprintf("jtree: invalid signature: %v", ft))
	}
	t := ft.Out(0)
	if t.Kind() != reflect.Interface {
		panic(fmt.Sprintf("jtree: user type must be an interface: %v", t))
	}
	r.mtx.Lock()
	defer r.mtx.Unlock()
	if _, ok := r.types[t]; ok {
		panic(fmt.Sprintf("jtree: duplicate user type: %v", t))
	}
	r.types[t] = fn
}

func (r *TypeRegistry) call(t reflect.Type, n Node, op []Option) (reflect.Value, error) {
	r.mtx.RLock()
	f, ok := r.types[t]
	r.mtx.RUnlock()
	if !ok {
		return reflect.Value{}, nil
	}
	out := reflect.ValueOf(f).Call([]reflect.Value{reflect.ValueOf(n), reflect.ValueOf(op)})
	if !out[1].IsNil() {
		return reflect.Value{}, out[1].Interface().(error)
	}
	return out[0], nil
}

func RegisterType(fn interface{}) {
	defaultRegistry.RegisterType(fn)
}

var defaultRegistry = NewTypeRegistry()
