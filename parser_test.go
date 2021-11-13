package jtree_test

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ecadlabs/jtree"
	"github.com/stretchr/testify/assert"
)

func newNumNode(s string) *jtree.Num {
	f, _, _ := new(big.Float).Parse(s, 10)
	return (*jtree.Num)(f)
}

func TestParseArray(t *testing.T) {
	src := []struct {
		s   string
		n   jtree.Node
		err string
	}{
		{s: `[123,"aaa","bbb"]`, n: jtree.Array{newNumNode("123"), jtree.String("aaa"), jtree.String("bbb")}},
		{s: `[123,"aaa","bbb",]`, n: jtree.Array{newNumNode("123"), jtree.String("aaa"), jtree.String("bbb")}},
		{s: `[]`, n: jtree.Array{}},
		{s: `[123,"aaa","bbb",`, err: "EOF"},
		{s: `[123,"aaa","bbb"`, err: "EOF"},
	}
	for _, s := range src {
		node, err := jtree.NewParser(strings.NewReader(s.s)).Parse()
		if s.err == "" {
			if assert.NoError(t, err) {
				assert.Equal(t, s.n, node)
			}
		} else {
			assert.EqualError(t, err, s.err)
		}
	}
}

func TestParseObject(t *testing.T) {
	src := []struct {
		s   string
		n   jtree.Node
		err string
	}{
		{
			s: `{"a":123,"b":"aaa","c":"bbb"}`,
			n: jtree.Fields{
				{"a", newNumNode("123")},
				{"b", jtree.String("aaa")},
				{"c", jtree.String("bbb")},
			}.NewObject(),
		},
		{
			s: `{"a":123,"b":"aaa","c":"bbb",}`,
			n: jtree.Fields{
				{"a", newNumNode("123")},
				{"b", jtree.String("aaa")},
				{"c", jtree.String("bbb")},
			}.NewObject(),
		},
		{s: `{}`, n: jtree.Fields{}.NewObject()},
		{s: `{"a":123,"b":"aaa","c":"bbb"`, err: "EOF"},
		{s: `{"a":123,"b":"aaa","c":`, err: "EOF"},
		{s: `{"a":123,"b":"aaa","c",`, err: "jtree: colon expected at position 22: ','"},
		{s: `{"a":123,"b":"aaa",123}`, err: "jtree: object key expected at position 19: '123'"},
	}
	for _, s := range src {
		node, err := jtree.NewParser(strings.NewReader(s.s)).Parse()
		if s.err == "" {
			if assert.NoError(t, err) {
				assert.Equal(t, s.n, node)
			}
		} else {
			assert.EqualError(t, err, s.err)
		}
	}
}
