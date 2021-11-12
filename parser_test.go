package jtree

import (
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newNumNode(s string) *Num {
	f, _, _ := new(big.Float).Parse(s, 10)
	return (*Num)(f)
}

func TestParseArray(t *testing.T) {
	src := []struct {
		s   string
		n   Node
		err string
	}{
		{s: `[123,"aaa","bbb"]`, n: Array{newNumNode("123"), String("aaa"), String("bbb")}},
		{s: `[123,"aaa","bbb",]`, n: Array{newNumNode("123"), String("aaa"), String("bbb")}},
		{s: `[]`, n: Array{}},
		{s: `[123,"aaa","bbb",`, err: "EOF"},
		{s: `[123,"aaa","bbb"`, err: "EOF"},
	}
	for _, s := range src {
		node, err := NewParser(strings.NewReader(s.s)).Parse()
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
		n   Node
		err string
	}{
		{s: `{"a":123,"b":"aaa","c":"bbb"}`, n: &Object{
			keys: []string{"a", "b", "c"},
			values: map[string]Node{
				"a": newNumNode("123"),
				"b": String("aaa"),
				"c": String("bbb"),
			},
		}},
		{s: `{"a":123,"b":"aaa","c":"bbb",}`, n: &Object{
			keys: []string{"a", "b", "c"},
			values: map[string]Node{
				"a": newNumNode("123"),
				"b": String("aaa"),
				"c": String("bbb"),
			},
		}},
		{s: `{}`, n: &Object{keys: make([]string, 0), values: make(map[string]Node)}},
		{s: `{"a":123,"b":"aaa","c":"bbb"`, err: "EOF"},
		{s: `{"a":123,"b":"aaa","c":`, err: "EOF"},
		{s: `{"a":123,"b":"aaa","c",`, err: "jtree: colon expected at position 22: ','"},
		{s: `{"a":123,"b":"aaa",123}`, err: "jtree: object key expected at position 19: '123'"},
	}
	for _, s := range src {
		node, err := NewParser(strings.NewReader(s.s)).Parse()
		if s.err == "" {
			if assert.NoError(t, err) {
				assert.Equal(t, s.n, node)
			}
		} else {
			assert.EqualError(t, err, s.err)
		}
	}
}
