package jtree_test

import (
	"errors"
	"fmt"
	"log"

	"github.com/ecadlabs/jtree"
)

type UserType interface {
	ImplKind() string
}

type UserTypeInt struct {
	Kind string `json:"kind"`
	Int  int    `json:"int"`
}

func (u *UserTypeInt) ImplKind() string { return "int" }

type UserTypeStr struct {
	Kind   string `json:"kind"`
	String string `json:"string"`
}

func (u *UserTypeStr) ImplKind() string { return "string" }

func UserTypeFunc(n jtree.Node, ctx *jtree.Context) (UserType, error) {
	obj, ok := n.(jtree.Object)
	if !ok {
		return nil, errors.New("object expected")
	}

	kind, ok := obj.FieldByName("kind").(jtree.String)
	if !ok {
		return nil, errors.New("malformed object")
	}

	var dest UserType
	switch kind {
	case "int":
		dest = new(UserTypeInt)
	case "string":
		dest = new(UserTypeStr)
	default:
		return nil, fmt.Errorf("unknown kind '%s'", string(kind))
	}

	var tmp interface{} = dest
	err := n.Decode(tmp, jtree.OpCtx(ctx))
	return dest, err
}

func Example_userInterfaceType() {
	src := `[
	{"kind": "int", "int": 123},
	{"kind": "string", "string": "text"},
]`
	var dest []UserType

	if err := jtree.Unmarshal([]byte(src), &dest); err != nil {
		log.Fatal(err)
	}

	for _, v := range dest {
		fmt.Printf("%s: %v\n", v.ImplKind(), v)
	}

	// Output:
	// int: &{int 123}
	// string: &{string text}
}

func init() {
	jtree.RegisterType(UserTypeFunc)
}
