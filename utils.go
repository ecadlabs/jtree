package jtree

import (
	"reflect"
	"strings"
)

type StructField struct {
	*reflect.StructField
	Options []string
	Name    string
}

func mkIndex(a, b []int) []int {
	cp := make([]int, len(a)+len(b))
	copy(cp, a)
	copy(cp[len(a):], b)
	return cp
}

func collectFields(t reflect.Type, index []int, ptr []reflect.Type, out map[string]*StructField) (list []*StructField) {
	list = make([]*StructField, 0)
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
			l := collectFields(ft, mkIndex(index, f.Index), ptr, out)
			list = append(list, l...)
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
			field := &StructField{
				StructField: &tmp,
				Options:     opt,
				Name:        name,
			}
			out[name] = field
			list = append(list, field)
		}
	}
	return
}

func VisibleFields(t reflect.Type) []*StructField {
	fields := make(map[string]*StructField)
	return collectFields(t, nil, nil, fields)
}

func parseTag(tag string) (name string, opt []string) {
	s := strings.Split(tag, ",")
	return s[0], s[1:]
}

func parseFieldOptions(tags []string, opt *options) []Option {
	out := make([]Option, 0, len(tags))
	elemOp := make([]Option, 0, len(tags))
	for _, s := range tags {
		if len(s) == 0 {
			continue
		}
		elem := false
		if s[0] == '[' {
			if s[len(s)-1] != ']' {
				continue
			}
			s = s[1 : len(s)-1]
			elem = true
		}
		var o Option
		if s == "string" {
			o = OpString
		} else if enc := opt.ctx().encodings().get(s); enc != nil {
			o = OpEncoding(enc)
		} else {
			continue
		}
		if elem {
			elemOp = append(elemOp, o)
			elem = false
		} else {
			out = append(out, o)
		}
	}
	if len(elemOp) != 0 {
		out = append(out, OpElem(elemOp...))
	}
	return out
}
