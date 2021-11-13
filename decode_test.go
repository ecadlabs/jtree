package jtree

import (
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

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
func newBool(b bool) *bool          { return &b }
func newUint(u uint) *uint          { return &u }
func newNode(n Node) *Node          { return &n }
func newFloat64(f float64) *float64 { return &f }
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
	objNode := Fields{
		{"F0", (*Num)(big.NewFloat(1))},
		{"f1", String("aaa")},
		{"f2", String("bbb")},
		{"f3", String("123")},
		{"f4", String("ccc")},
		{"S", (*Num)(big.NewFloat(2))},
		{"ZZ", Null{}},
		{"XX", (*Num)(big.NewFloat(3))},
	}.NewObject()

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
	}

	tst := []struct {
		n      Node
		out    interface{}
		expect interface{}
		op     []Option
		err    string
	}{
		{n: String("aaa"), out: new(string), expect: newStr("aaa")},
		{n: String("aaa"), out: new(*string), expect: newStrP("aaa")},
		{n: String("aaa"), out: new([]byte), expect: newBytes("aaa"), op: []Option{OpString}},
		{n: String("YWFh"), out: new([]byte), expect: newBytes("aaa")},
		{n: String("YWFh"), out: new(string), expect: newStr("aaa"), op: []Option{OpEncoding(Base64)}},
		{n: String("616161"), out: new([]byte), expect: newBytes("aaa"), op: []Option{OpEncoding(Hex)}},
		{n: String("aaa"), out: new(Node), expect: newNode(String("aaa"))},
		{n: String("123"), out: new(int), expect: newInt(123), op: []Option{OpString}},
		{n: String("123"), out: new(*int), expect: newIntP(123), op: []Option{OpString}},
		{n: String("123"), out: new(int), err: "jtree: can't convert string to int"},
		{n: String("123"), out: new(uint), expect: newUint(123), op: []Option{OpString}},
		{n: String("123"), out: new(big.Int), expect: big.NewInt(123)},
		{n: String("123"), out: new(big.Float), expect: newBigFloat("123")},
		{n: String("true"), out: new(bool), expect: newBool(true), op: []Option{OpString}},
		{n: String("true"), out: new(bool), err: "jtree: can't convert string to bool"},
		{n: String("zzz"), out: new(bool), op: []Option{OpString}, err: "jtree: strconv.ParseBool: parsing \"zzz\": invalid syntax"},
		{n: String("yep"), out: new(CanDecode), expect: (*CanDecode)(newInt(1))},
		{n: String("nope"), out: new(CanDecode), expect: (*CanDecode)(newInt(0))},
		{n: String("maybe"), out: new(CanDecode), expect: (*CanDecode)(newInt(-1))},
		{n: String("whaaat"), out: new(CanDecode), err: "jtree: unknown string: whaaat"},
		{n: newNumNode("123"), out: new(int), expect: newInt(123)},
		{n: newNumNode("123"), out: new(*int), expect: newIntP(123)},
		{n: newNumNode("123"), out: new(uint), expect: newUint(123)},
		{n: newNumNode("123"), out: new(string), expect: newStr("123")},
		{n: newNumNode("123"), out: new(float64), expect: newFloat64(123)},
		{n: newNumNode("123"), out: new(big.Float), expect: newBigFloat("123")},
		{n: newNumNode("123"), out: new(big.Int), expect: newBigInt("123")},
		{n: newNumNode("1"), out: new(bool), expect: newBool(true)},
		{n: newNumNode("0"), out: new(bool), expect: newBool(false)},

		{n: String("2021-11-11T15:08:52.537Z"), out: new(time.Time), expect: mkTime("2021-11-11T15:08:52.537Z")},
		{n: newNumNode("1636643332"), out: new(time.Time), expect: mkTime("2021-11-11T15:08:52Z")},

		{n: Null{}, out: newInt(123), expect: newInt(0)},
		{n: Null{}, out: newIntP(120), expect: new(*int)},
		{n: Null{}, out: new(interface{}), expect: new(interface{})},

		{n: Array{String("aaa"), String("bbb")}, out: new([]string), expect: &[]string{"aaa", "bbb"}},
		{n: Array{String("aaa"), String("bbb")}, out: new([3]string), expect: &[3]string{"aaa", "bbb"}},
		{n: Array{String("aaa"), String("bbb")}, out: new([1]string), expect: &[1]string{"aaa"}},

		{
			n: Fields{
				{"f0", (*Num)(big.NewFloat(1))},
				{"f1", String("aaa")},
				{"f2", String("ccc")},
			}.NewObject(),
			out: new(map[string]interface{}),
			expect: &(map[string]interface{}{
				"f0": float64(1),
				"f1": "aaa",
				"f2": "ccc",
			}),
		},
		{
			n: Fields{
				{"f0", (*Num)(big.NewFloat(1))},
				{"f1", (*Num)(big.NewFloat(2))},
				{"f2", (*Num)(big.NewFloat(3))},
			}.NewObject(),
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
			err: "jtree: undefined field 'S': jtree.T0",
			op:  []Option{OpDisallowUnknownFields},
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
		n      Node
		expect interface{}
	}{
		{n: newNumNode("123"), expect: float64(123)},
		{n: String("aaa"), expect: "aaa"},
		{n: Array{String("aaa"), String("bbb")}, expect: []interface{}{"aaa", "bbb"}},
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

type UserType interface {
	ImplKind() string
}

type UserTypeInt struct {
	Kind string `json:"kind"`
	Int  int    `json:"int"`
}

func (u *UserTypeInt) ImplKind() string { return "int" }

type UserTypeStr struct {
	Kind string `json:"kind"`
	Str  string `json:"str"`
}

func (u *UserTypeStr) ImplKind() string { return "str" }

func UserTypeFunc(n Node, op []Option) (UserType, error) {
	obj, ok := n.(*Object)
	if !ok {
		return nil, errors.New("not an object")
	}
	kind, ok := obj.FieldByName("kind").(String)
	if !ok {
		return nil, errors.New("malformed object")
	}
	var dest UserType
	switch kind {
	case "int":
		dest = &UserTypeInt{}
	case "str":
		dest = &UserTypeStr{}
	default:
		return nil, fmt.Errorf("unknown kind '%s'", string(kind))
	}
	return dest, n.Decode(dest, OpGlobal(op))
}

func TestUserType(t *testing.T) {
	tst := []struct {
		n      Node
		out    *UserType
		expect UserType
		err    string
	}{
		{
			n: Fields{
				{"kind", String("int")},
				{"int", (*Num)(big.NewFloat(1))},
			}.NewObject(),
			out: new(UserType),
			expect: &UserTypeInt{
				Kind: "int",
				Int:  1,
			},
		},
		{
			n: Fields{
				{"kind", String("str")},
				{"str", String("aaa")},
			}.NewObject(),
			out: new(UserType),
			expect: &UserTypeStr{
				Kind: "str",
				Str:  "aaa",
			},
		},
	}

	reg := NewTypeRegistry()
	reg.RegisterType(UserTypeFunc)

	for _, tt := range tst {
		err := tt.n.Decode(tt.out, OpTypes(reg))
		if tt.err != "" {
			assert.EqualError(t, err, tt.err)
		} else if assert.NoError(t, err) {
			assert.Equal(t, &tt.expect, tt.out)
		}
	}
}

type CanDecode int

func (c *CanDecode) DecodeJSON(node Node) error {
	if s, ok := node.(String); ok {
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
