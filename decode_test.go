package jtree

import (
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestString(t *testing.T) {
	tst := []struct {
		n      String
		out    interface{}
		expect interface{}
		op     []Option
		err    string
	}{
		{n: "aaa", out: new(string), expect: newStr("aaa")},
		{n: "aaa", out: new(*string), expect: newStrP("aaa")},
		{n: "aaa", out: new([]byte), expect: newBytes("aaa"), op: []Option{OpString}},
		{n: "YWFh", out: new([]byte), expect: newBytes("aaa")},
		{n: "YWFh", out: new(string), expect: newStr("aaa"), op: []Option{OpEncoding(Base64)}},
		{n: "616161", out: new([]byte), expect: newBytes("aaa"), op: []Option{OpEncoding(Hex)}},
		{n: "aaa", out: new(Node), expect: newNode(String("aaa"))},
		{n: "123", out: new(int), expect: newInt(123), op: []Option{OpString}},
		{n: "123", out: new(*int), expect: newIntP(123), op: []Option{OpString}},
		{n: "123", out: new(int), err: "jtree: can't convert string to int"},
		{n: "123", out: new(uint), expect: newUint(123), op: []Option{OpString}},
		{n: "123", out: new(big.Int), expect: big.NewInt(123)},
		{n: "123", out: new(big.Float), expect: newBigFloat("123")},
		{n: "true", out: new(bool), expect: newBool(true), op: []Option{OpString}},
		{n: "true", out: new(bool), err: "jtree: can't convert string to bool"},
		{n: "zzz", out: new(bool), op: []Option{OpString}, err: "jtree: strconv.ParseBool: parsing \"zzz\": invalid syntax"},
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

func TestTextUnmarshaler(t *testing.T) {
	var v time.Time
	require.NoError(t, String("2021-11-11T15:08:52.537Z").Decode(&v))
	var expect time.Time
	expect.UnmarshalText([]byte("2021-11-11T15:08:52.537Z"))
	require.Equal(t, expect, v)
}

func TestNum(t *testing.T) {
	tst := []struct {
		n      *Num
		out    interface{}
		expect interface{}
	}{
		{n: newNumNode("123"), out: new(int), expect: newInt(123)},
		{n: newNumNode("123"), out: new(*int), expect: newIntP(123)},
		{n: newNumNode("123"), out: new(uint), expect: newUint(123)},
		{n: newNumNode("123"), out: new(string), expect: newStr("123")},
		{n: newNumNode("123"), out: new(float64), expect: newFloat64(123)},
		{n: newNumNode("123"), out: new(big.Float), expect: newBigFloat("123")},
		{n: newNumNode("123"), out: new(big.Int), expect: newBigInt("123")},
		{n: newNumNode("1"), out: new(bool), expect: newBool(true)},
		{n: newNumNode("0"), out: new(bool), expect: newBool(false)},
	}
	for _, tt := range tst {
		if assert.NoError(t, tt.n.Decode(tt.out)) {
			assert.Equal(t, tt.expect, tt.out)
		}
	}
}

func TestNumToTime(t *testing.T) {
	var v time.Time
	require.NoError(t, (*Num)(big.NewFloat(1636643332)).Decode(&v))
	var expect time.Time
	expect.UnmarshalText([]byte("2021-11-11T15:08:52Z"))
	require.Equal(t, expect, v)
}

func TestNull(t *testing.T) {
	tst := []struct {
		out    interface{}
		expect interface{}
	}{
		{out: newInt(123), expect: newInt(0)},
		{out: newIntP(120), expect: new(*int)},
		{out: new(interface{}), expect: new(interface{})},
	}
	for _, tt := range tst {
		if assert.NoError(t, Null{}.Decode(tt.out)) {
			assert.Equal(t, tt.expect, tt.out)
		}
	}
}

func TestNullify(t *testing.T) {
	var i int
	ptr := &i
	require.NoError(t, Null{}.Decode(&ptr))
	require.Nil(t, ptr)
}

func TestDefault(t *testing.T) {
	tst := []struct {
		n      Node
		expect interface{}
	}{
		{n: newNumNode("123"), expect: float64(123)},
		{n: String("aaa"), expect: "aaa"},
	}
	for _, tt := range tst {
		var dest interface{}
		if assert.NoError(t, tt.n.Decode(&dest)) {
			assert.Equal(t, tt.expect, dest)
		}
	}
}

func TestArray(t *testing.T) {
	tst := []struct {
		n      Array
		out    interface{}
		expect interface{}
	}{
		{n: Array{String("aaa"), String("bbb")}, out: new([]string), expect: &[]string{"aaa", "bbb"}},
		{n: Array{String("aaa"), String("bbb")}, out: new([3]string), expect: &[3]string{"aaa", "bbb"}},
		{n: Array{String("aaa"), String("bbb")}, out: new([1]string), expect: &[1]string{"aaa"}},
	}
	for _, tt := range tst {
		if assert.NoError(t, tt.n.Decode(tt.out)) {
			assert.Equal(t, tt.expect, tt.out)
		}
	}
}

func TestArrayDefault(t *testing.T) {
	src := Array{String("aaa"), String("bbb")}
	var v interface{}
	require.NoError(t, src.Decode(&v))
	require.Equal(t, []interface{}{"aaa", "bbb"}, v)
}

type T0 struct {
	T1
	F0 int
	F1 string `json:"f1"`
	*T2
	FF *string
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

func TestObject(t *testing.T) {
	src := Fields{
		{"F0", (*Num)(big.NewFloat(1))},
		{"f1", String("aaa")},
		{"f2", String("bbb")},
		{"f3", String("123")},
		{"f4", String("ccc")},
		{"S", (*Num)(big.NewFloat(2))},
		{"ZZ", Null{}},
	}
	expect := T0{
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
	var dst0 T0
	if assert.NoError(t, src.NewObject().Decode(&dst0)) {
		assert.Equal(t, expect, dst0)
	}
	var dst1 T0
	err := src.NewObject().Decode(&dst1, OpDisallowUnknownFields)
	assert.EqualError(t, err, "jtree: undefined field 'S': jtree.T0")

	var dst2 *T0
	if assert.NoError(t, src.NewObject().Decode(&dst2)) {
		assert.Equal(t, &expect, dst2)
	}
}

func TestMapInterface(t *testing.T) {
	src := Fields{
		{"f0", (*Num)(big.NewFloat(1))},
		{"f1", String("aaa")},
		{"f2", String("ccc")},
	}
	var dst0 map[string]interface{}
	if assert.NoError(t, src.NewObject().Decode(&dst0)) {
		assert.Equal(t, map[string]interface{}{
			"f0": float64(1),
			"f1": "aaa",
			"f2": "ccc",
		}, dst0)
	}
}

func TestMapInt(t *testing.T) {
	src := Fields{
		{"f0", (*Num)(big.NewFloat(1))},
		{"f1", (*Num)(big.NewFloat(2))},
		{"f2", (*Num)(big.NewFloat(3))},
	}
	var dst0 map[string]int
	if assert.NoError(t, src.NewObject().Decode(&dst0)) {
		assert.Equal(t, map[string]int{
			"f0": 1,
			"f1": 2,
			"f2": 3,
		}, dst0)
	}
}

func TestMapDefault(t *testing.T) {
	src := Fields{
		{"f0", (*Num)(big.NewFloat(1))},
		{"f1", (*Num)(big.NewFloat(2))},
		{"f2", (*Num)(big.NewFloat(3))},
	}
	var dst0 interface{}
	if assert.NoError(t, src.NewObject().Decode(&dst0)) {
		assert.Equal(t, map[string]interface{}{
			"f0": float64(1),
			"f1": float64(2),
			"f2": float64(3),
		}, dst0)
	}
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

func TestUserType(t *testing.T) {
	fn := func(n Node, op []Option) (UserType, error) {
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
	reg := NewTypeRegistry()
	reg.RegisterType(fn)

	var dest0 UserType
	src0 := Fields{
		{"kind", String("int")},
		{"int", (*Num)(big.NewFloat(1))},
	}
	if assert.NoError(t, src0.NewObject().Decode(&dest0, OpTypes(reg))) {
		assert.Equal(t, &UserTypeInt{
			Kind: "int",
			Int:  1,
		}, dest0)
	}

	var dest1 UserType
	src1 := Fields{
		{"kind", String("str")},
		{"str", String("aaa")},
	}
	if assert.NoError(t, src1.NewObject().Decode(&dest1, OpTypes(reg))) {
		assert.Equal(t, &UserTypeStr{
			Kind: "str",
			Str:  "aaa",
		}, dest1)
	}
}
