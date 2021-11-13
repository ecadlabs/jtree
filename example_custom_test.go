package jtree_test

import (
	"bytes"
	"errors"
	"fmt"
	"log"

	"github.com/ecadlabs/jtree"
)

type Cutie int

const (
	Snek Cutie = iota
	Pupper
	Froggo
)

func (c *Cutie) DecodeJSON(node jtree.Node) error {
	name, ok := node.(jtree.String)
	if !ok {
		return errors.New("string expected")
	}
	switch name {
	case "snek":
		*c = Snek
	case "pupper":
		*c = Pupper
	case "froggo":
		*c = Froggo
	default:
		return fmt.Errorf("unknown kind of cutie: %s", name)
	}
	return nil
}

func Example_customDecoder() {
	src := `["snek","pupper","froggo"]`

	var dest []Cutie
	parser := jtree.NewParser(bytes.NewReader([]byte(src)))
	node, err := parser.Parse()
	if err != nil {
		log.Fatal(err)
	}
	if err := node.Decode(&dest); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%v\n", dest)

	// Output:
	// [0 1 2]
}
