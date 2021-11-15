package jtree_test

import (
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ecadlabs/jtree"
	"github.com/stretchr/testify/assert"
)

func newStr(s string) *string { return &s }
func newStrP(s string) **string {
	p := &s
	return &p
}
func newBytes(s string) *[]byte {
	b := []byte(s)
	return &b
}

func newInt(s int) *int { return &s }
func newIntP(i int) **int {
	p := &i
	return &p
}
func newBool(b bool) *bool             { return &b }
func newUint(u uint) *uint             { return &u }
func newNode(n jtree.Node) *jtree.Node { return &n }
func newFloat64(f float64) *float64    { return &f }
func newBigFloat(s string) *big.Float {
	f, _, _ := new(big.Float).Parse(s, 10)
	return f
}
func newBigInt(s string) *big.Int {
	i, _ := new(big.Int).SetString(s, 10)
	return i
}

func mkTime(s string) *time.Time {
	var t time.Time
	t.UnmarshalText([]byte(s))
	return &t
}

func TestDecode(t *testing.T) {
	objNode := jtree.Object{
		{"F0", (*jtree.Num)(big.NewFloat(1))},
		{"f1", jtree.String("aaa")},
		{"f2", jtree.String("bbb")},
		{"f3", jtree.String("123")},
		{"f4", jtree.String("ccc")},
		{"S", (*jtree.Num)(big.NewFloat(2))},
		{"ZZ", jtree.Null{}},
		{"XX", (*jtree.Num)(big.NewFloat(3))},
		{"QQ", jtree.Array{jtree.String("123"), jtree.String("456")}},
	}

	objVal := &T0{
		T1: T1{
			F0: 0,
			F2: "bbb",
		},
		F0: 1,
		F1: "aaa",
		T2: &T2{
			F3: 123,
			F4: "ccc",
			S:  0,
		},
		QQ: []int{123, 456},
	}

	tst := []struct {
		n      jtree.Node
		out    interface{}
		expect interface{}
		op     []jtree.Option
		err    string
	}{
		{n: jtree.String("aaa"), out: new(string), expect: newStr("aaa")},
		{n: jtree.String("aaa"), out: new(*string), expect: newStrP("aaa")},
		{n: jtree.String("aaa"), out: new([]byte), expect: newBytes("aaa"), op: []jtree.Option{jtree.OpString}},
		{n: jtree.String("YWFh"), out: new([]byte), expect: newBytes("aaa")},
		{n: jtree.String("YWFh"), out: new(string), expect: newStr("aaa"), op: []jtree.Option{jtree.OpEncoding(jtree.Base64)}},
		{n: jtree.String("616161"), out: new([]byte), expect: newBytes("aaa"), op: []jtree.Option{jtree.OpEncoding(jtree.Hex)}},
		{n: jtree.String("aaa"), out: new(jtree.Node), expect: newNode(jtree.String("aaa"))},
		{n: jtree.String("123"), out: new(int), expect: newInt(123), op: []jtree.Option{jtree.OpString}},
		{n: jtree.String("123"), out: new(*int), expect: newIntP(123), op: []jtree.Option{jtree.OpString}},
		{n: jtree.String("123"), out: new(int), err: "jtree: can't convert string to int"},
		{n: jtree.String("123"), out: new(uint), expect: newUint(123), op: []jtree.Option{jtree.OpString}},
		{n: jtree.String("123"), out: new(big.Int), expect: big.NewInt(123)},
		{n: jtree.String("123"), out: new(big.Float), expect: newBigFloat("123")},
		{n: jtree.String("true"), out: new(bool), expect: newBool(true), op: []jtree.Option{jtree.OpString}},
		{n: jtree.String("true"), out: new(bool), err: "jtree: can't convert string to bool"},
		{n: jtree.String("zzz"), out: new(bool), op: []jtree.Option{jtree.OpString}, err: "jtree: strconv.ParseBool: parsing \"zzz\": invalid syntax"},
		{n: jtree.String("yep"), out: new(CanDecode), expect: (*CanDecode)(newInt(1))},
		{n: jtree.String("nope"), out: new(CanDecode), expect: (*CanDecode)(newInt(0))},
		{n: jtree.String("maybe"), out: new(CanDecode), expect: (*CanDecode)(newInt(-1))},
		{n: jtree.String("whaaat"), out: new(CanDecode), err: "unknown string: whaaat"},
		{n: newNumNode("123"), out: new(int), expect: newInt(123)},
		{n: newNumNode("123"), out: new(*int), expect: newIntP(123)},
		{n: newNumNode("123"), out: new(uint), expect: newUint(123)},
		{n: newNumNode("123"), out: new(string), expect: newStr("123")},
		{n: newNumNode("123"), out: new(float64), expect: newFloat64(123)},
		{n: newNumNode("123"), out: new(big.Float), expect: newBigFloat("123")},
		{n: newNumNode("123"), out: new(big.Int), expect: newBigInt("123")},
		{n: newNumNode("1"), out: new(bool), expect: newBool(true)},
		{n: newNumNode("0"), out: new(bool), expect: newBool(false)},

		{n: jtree.String("2021-11-11T15:08:52.537Z"), out: new(time.Time), expect: mkTime("2021-11-11T15:08:52.537Z")},
		{n: newNumNode("1636643332"), out: new(time.Time), expect: mkTime("2021-11-11T15:08:52Z")},

		{n: jtree.Null{}, out: newInt(123), expect: newInt(0)},
		{n: jtree.Null{}, out: newIntP(120), expect: new(*int)},
		{n: jtree.Null{}, out: new(interface{}), expect: new(interface{})},

		{n: jtree.Array{jtree.String("aaa"), jtree.String("bbb")}, out: new([]string), expect: &[]string{"aaa", "bbb"}},
		{n: jtree.Array{jtree.String("aaa"), jtree.String("bbb")}, out: new([3]string), expect: &[3]string{"aaa", "bbb"}},
		{n: jtree.Array{jtree.String("aaa"), jtree.String("bbb")}, out: new([1]string), expect: &[1]string{"aaa"}},

		{n: jtree.Array{jtree.String("123"), jtree.String("456")}, out: new([]int), expect: &[]int{123, 456}, op: []jtree.Option{jtree.OpElem(jtree.OpString)}},

		{
			n: jtree.Object{
				{"f0", (*jtree.Num)(big.NewFloat(1))},
				{"f1", jtree.String("aaa")},
				{"f2", jtree.String("ccc")},
			},
			out: new(map[string]interface{}),
			expect: &(map[string]interface{}{
				"f0": float64(1),
				"f1": "aaa",
				"f2": "ccc",
			}),
		},
		{
			n: jtree.Object{
				{"f0", (*jtree.Num)(big.NewFloat(1))},
				{"f1", (*jtree.Num)(big.NewFloat(2))},
				{"f2", (*jtree.Num)(big.NewFloat(3))},
			},
			out: new(map[string]int),
			expect: &(map[string]int{
				"f0": 1,
				"f1": 2,
				"f2": 3,
			}),
		},
		{
			n:      objNode,
			out:    new(T0),
			expect: objVal,
		},
		{
			n:      objNode,
			out:    new(*T0),
			expect: &objVal,
		},
		{
			n:   objNode,
			out: new(T0),
			err: "jtree: undefined field 'S': jtree_test.T0",
			op:  []jtree.Option{jtree.OpDisallowUnknownFields},
		},
	}
	for _, tt := range tst {
		err := tt.n.Decode(tt.out, tt.op...)
		if tt.err != "" {
			assert.EqualError(t, err, tt.err)
		} else if assert.NoError(t, err) {
			assert.Equal(t, tt.expect, tt.out)
		}
	}
}

func TestInterface(t *testing.T) {
	tst := []struct {
		n      jtree.Node
		expect interface{}
	}{
		{n: newNumNode("123"), expect: float64(123)},
		{n: jtree.String("aaa"), expect: "aaa"},
		{n: jtree.Array{jtree.String("aaa"), jtree.String("bbb")}, expect: []interface{}{"aaa", "bbb"}},
	}
	for _, tt := range tst {
		var dest interface{}
		if assert.NoError(t, tt.n.Decode(&dest)) {
			assert.Equal(t, tt.expect, dest)
		}
	}
}

type T0 struct {
	T1
	F0 int
	F1 string `json:"f1"`
	*T2
	FF *string
	*unexported
	QQ []int `json:",[string]"`
}

type T1 struct {
	F0 int
	F2 string `json:"f2"`
}

type T2 struct {
	F3 int    `json:"f3,string"`
	F4 string `json:"f4"`
	S  int    `json:"-"`
	ZZ *string
}

type unexported struct {
	XX int
}

type userType interface {
	ImplKind() string
}

type userTypeInt struct {
	Kind string `json:"kind"`
	Int  int    `json:"int"`
}

func (u *userTypeInt) ImplKind() string { return "int" }

type userTypeStr struct {
	Kind string `json:"kind"`
	Str  string `json:"str"`
}

func (u *userTypeStr) ImplKind() string { return "str" }

func userTypeFunc(n jtree.Node, ctx *jtree.Context) (userType, error) {
	obj, ok := n.(jtree.Object)
	if !ok {
		return nil, errors.New("not an object")
	}
	kind, ok := obj.FieldByName("kind").(jtree.String)
	if !ok {
		return nil, errors.New("malformed object")
	}
	var dest userType
	switch kind {
	case "int":
		dest = &userTypeInt{}
	case "str":
		dest = &userTypeStr{}
	default:
		return nil, fmt.Errorf("unknown kind '%s'", string(kind))
	}
	return dest, n.Decode(dest, jtree.OpCtx(ctx))
}

func TestUserType(t *testing.T) {
	tst := []struct {
		n      jtree.Node
		out    *userType
		expect userType
		err    string
	}{
		{
			n: jtree.Object{
				{"kind", jtree.String("int")},
				{"int", (*jtree.Num)(big.NewFloat(1))},
			},
			out: new(userType),
			expect: &userTypeInt{
				Kind: "int",
				Int:  1,
			},
		},
		{
			n: jtree.Object{
				{"kind", jtree.String("str")},
				{"str", jtree.String("aaa")},
			},
			out: new(userType),
			expect: &userTypeStr{
				Kind: "str",
				Str:  "aaa",
			},
		},
	}

	reg := jtree.NewTypeRegistry()
	reg.RegisterType(userTypeFunc)

	for _, tt := range tst {
		err := tt.n.Decode(tt.out, jtree.OpTypes(reg))
		if tt.err != "" {
			assert.EqualError(t, err, tt.err)
		} else if assert.NoError(t, err) {
			assert.Equal(t, &tt.expect, tt.out)
		}
	}
}

type CanDecode int

func (c *CanDecode) DecodeJSON(node jtree.Node) error {
	if s, ok := node.(jtree.String); ok {
		switch s {
		case "nope":
			*c = 0
		case "yep":
			*c = 1
		case "maybe":
			*c = -1
		default:
			return fmt.Errorf("unknown string: %s", s)
		}
		return nil
	}
	return fmt.Errorf("string expected: %s", node.Type())
}
