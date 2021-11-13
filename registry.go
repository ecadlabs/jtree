package jtree

import (
	"fmt"
	"reflect"
	"sync"
)

// TypeRegistry stores uses interface type constructors (decoders)
type TypeRegistry struct {
	types map[reflect.Type]interface{}
	mtx   sync.RWMutex
}

// NewTypeRegistry returns new empty TypeRegistry
func NewTypeRegistry() *TypeRegistry {
	return &TypeRegistry{
		types: make(map[reflect.Type]interface{}),
	}
}

var ctxType = reflect.TypeOf((*Context)(nil))

// RegisterType registers user interface type. The argument is a constructor function of type `func(Node, []Option) (UserType, error)`.
// It panics if any other type is passed
func (r *TypeRegistry) RegisterType(fn interface{}) {
	ft := reflect.TypeOf(fn)
	if ft.Kind() != reflect.Func {
		panic(fmt.Sprintf("jtree: function expected: %v", ft))
	}
	if ft.NumIn() != 2 || ft.In(0) != nodeType || ft.In(1) != ctxType || ft.NumOut() != 2 || ft.Out(1) != errorType {
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

func (r *TypeRegistry) call(t reflect.Type, n Node, ctx *Context) (reflect.Value, error) {
	r.mtx.RLock()
	f, ok := r.types[t]
	r.mtx.RUnlock()
	if !ok {
		return reflect.Value{}, nil
	}
	out := reflect.ValueOf(f).Call([]reflect.Value{reflect.ValueOf(n), reflect.ValueOf(ctx)})
	if !out[1].IsNil() {
		return reflect.Value{}, out[1].Interface().(error)
	}
	return out[0], nil
}

// RegisterType registers user interface type in the global registry
func RegisterType(fn interface{}) {
	defaultTypeRegistry.RegisterType(fn)
}

// EncodingRegistry stores user encoding schemes
type EncodingRegistry struct {
	encodings map[string]Encoding
	mtx       sync.RWMutex
}

// NewEncodingRegistry returns new empty EncodingRegistry
func NewEncodingRegistry() *EncodingRegistry {
	return &EncodingRegistry{
		encodings: make(map[string]Encoding),
	}
}

// RegisterEncoding registers custom encoding scheme under provided name
func (r *EncodingRegistry) RegisterEncoding(name string, enc Encoding) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	if _, ok := r.encodings[name]; ok {
		panic(fmt.Sprintf("jtree: duplicate encoding: %v", name))
	}
	r.encodings[name] = enc
}

func (r *EncodingRegistry) get(name string) Encoding {
	r.mtx.RLock()
	e := r.encodings[name]
	r.mtx.RUnlock()
	return e
}

// RegisterEncoding registers custom encoding scheme under provided name in the global registry
func RegisterEncoding(name string, enc Encoding) {
	defaultEncodingRegistry.RegisterEncoding(name, enc)
}

var defaultTypeRegistry = NewTypeRegistry()
var defaultEncodingRegistry = NewEncodingRegistry()

func init() {
	RegisterEncoding("base64", Base64)
	RegisterEncoding("hex", Hex)
}
