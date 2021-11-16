// Package jtree is the AST centered JSON parser
package jtree

import (
	"encoding"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"time"
)

// Context stores global options
type Context struct {
	noUnknown bool
	typeReg   *TypeRegistry
	encReg    *EncodingRegistry
}

func (c *Context) types() *TypeRegistry {
	if c.typeReg != nil {
		return c.typeReg
	}
	return defaultTypeRegistry
}

func (c *Context) encodings() *EncodingRegistry {
	if c.encReg != nil {
		return c.encReg
	}
	return defaultEncodingRegistry
}

type options struct {
	context *Context
	str     bool
	enc     Encoding
	elem    *options
}

func (o *options) apply(opts []Option) *options {
	for _, fn := range opts {
		fn(o)
	}
	return o
}

func (o *options) ctx() *Context {
	if o.context != nil {
		return o.context
	}
	o.context = new(Context)
	return o.context
}

// OpString makes the numeric value to be encoded/decoded as a string and the byte slice value
// to be converted to a string as is (skips the binary encoding scheme)
func OpString(o *options) { o.str = true }

// OpEncoding specifies the binary encoding scheme used for byte slices. Without this option base64 scheme will be used
func OpEncoding(e Encoding) Option { return func(o *options) { o.enc = e } }

// OpTypes provides custom user type registry. The option is global for all Decode calls in chain
func OpTypes(r *TypeRegistry) Option { return func(o *options) { o.ctx().typeReg = r } }

// OpEncodings provides custom user encodings registry. The option is global for all Decode calls in chain
func OpEncodings(e *EncodingRegistry) Option { return func(o *options) { o.ctx().encReg = e } }

// OpDisallowUnknownFields causes the Decode method to return an error when the destination is a struct
// and the input contains object keys which do not match any non-ignored, exported fields in the destination.
func OpDisallowUnknownFields(o *options) { o.ctx().noUnknown = true }

// OpElem passes options to container elements
func OpElem(op ...Option) Option {
	return func(o *options) {
		if o.elem == nil {
			o.elem = new(options)
		}
		o.elem.apply(op)
	}
}

// OpCtx passes global options to subsequent Decode calls. Used in custom decoders
func OpCtx(ctx *Context) Option { return func(o *options) { o.context = ctx } }

func opInit(src *options) Option {
	return func(o *options) {
		*o = *src
		o.elem = nil
	}
}

// Option is the function pointer used to pass options to Decode method
type Option func(*options)

// Node is the JSON AST node
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

// JSONDecoder is the interface implemented by types that can decode a JSON description of themselves.
type JSONDecoder interface {
	DecodeJSON(node Node) error
}

// Num represents numeric node
type Num big.Float // on conversion operations the difference in performance between big.Float and big.Int is insignificant

// Type returns the node type i.e. "number"
func (*Num) Type() string { return "number" }

// Decode decodes the node into the value pointed by v
func (n *Num) Decode(v interface{}, op ...Option) error {
	fn := func(out reflect.Value, opt *options) error {
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
	fn := func(out reflect.Value, opt *options) error {
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
type Object []*Field

// Type returns the node i.e. "object"
func (Object) Type() string { return "object" }

// Field is used to construct objects
type Field struct {
	Key   string
	Value Node
}

// Keys returns all object keys
func (o Object) Keys() []string {
	keys := make([]string, len(o))
	for i, f := range o {
		keys[i] = f.Key
	}
	return keys
}

// FieldByName returns the field with specific name or nil
func (o Object) FieldByName(field string) Node {
	for _, f := range o {
		if f.Key == field {
			return f.Value
		}
	}
	return nil
}

// Field returns i'th field or nil if the number of fields is exceeded
func (o Object) Field(i int) (string, Node) {
	if i >= len(o) {
		return "", nil
	}
	return o[i].Key, o[i].Value
}

// NumField returns the number of fields
func (o Object) NumField() int {
	return len(o)
}

// Decode decodes the node into the value pointed by v
func (o Object) Decode(v interface{}, op ...Option) error {
	fn := func(out reflect.Value, opt *options) error {
		t := out.Type()
		switch t.Kind() {
		case reflect.Struct:
			fields := make(map[string]*StructField)
			collectFields(t, nil, nil, fields)
			for i := 0; i < o.NumField(); i++ {
				key, elem := o.Field(i)
				field, ok := fields[key]
				if !ok {
					if opt.ctx().noUnknown {
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
				fopt := parseFieldOptions(field.Options, opt)
				if err := elem.Decode(dest.Addr().Interface(), mkChildOptions(opt, fopt)...); err != nil {
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
				if err := elem.Decode(elemVal.Interface(), mkChildOptions(opt, nil)...); err != nil {
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
	fn := func(out reflect.Value, opt *options) error {
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
			if err := elem.Decode(dst.Index(i).Addr().Interface(), mkChildOptions(opt, nil)...); err != nil {
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

// Bool represents boolean node
type Bool bool

// Type returns the node i.e. "boolean"
func (Bool) Type() string { return "boolean" }

// Decode decodes the node into the value pointed by v
func (b Bool) Decode(v interface{}, op ...Option) error {
	fn := func(out reflect.Value, opt *options) error {
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

// Null represents null node
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
	decoderType         = reflect.TypeOf((*JSONDecoder)(nil)).Elem()
)

type decodeFunc func(out reflect.Value, opt *options) error

func decodeNode(v interface{}, node Node, decode decodeFunc, op ...Option) error {
	opt := new(options).apply(op)
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

	// concrete type
	if out.Kind() != reflect.Interface {
		if reflect.PtrTo(out.Type()).Implements(decoderType) && out.CanAddr() {
			dec := out.Addr().Interface().(JSONDecoder)
			if err := dec.DecodeJSON(node); err != nil {
				return err
			}
			return nil
		}
		return decode(out, opt)
	}

	if out.Type() == nodeType {
		// special case
		out.Set(reflect.ValueOf(node))
		return nil
	}

	// user interface type
	val, err := opt.ctx().types().call(out.Type(), node, opt.context)
	if err != nil {
		return err
	}
	if val.IsValid() {
		out.Set(val)
		return nil
	}

	// allocate default type
	var dst reflect.Value
	switch node.(type) {
	case *Num:
		dst = reflect.New(float64Type).Elem()
	case String:
		dst = reflect.New(stringType).Elem()
	case Object:
		dst = reflect.New(objectType).Elem()
	case Array:
		dst = reflect.New(arrayType).Elem()
	case Bool:
		dst = reflect.New(boolType).Elem()
	default:
		panic("unknown node")
	}
	if err := decode(dst, opt); err != nil {
		return err
	}
	if !dst.CanConvert(out.Type()) {
		return fmt.Errorf("jtree: can't convert %v to %v", dst.Type(), out.Type())
	}
	out.Set(dst.Convert(out.Type()))
	return nil
}

func mkChildOptions(opt *options, fopt []Option) []Option {
	out := make([]Option, 0, len(fopt)+2)
	if opt.elem != nil {
		out = append(out, opInit(opt.elem))
	}
	out = append(out, OpCtx(opt.context))
	return append(out, fopt...)
}
