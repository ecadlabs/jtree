package jtree

import (
	"fmt"
	"io"
	"math/big"
)

// Parser parses JSON stream into an AST representation
type Parser struct {
	r *reader
}

// NewParser returns new Parser
func NewParser(r io.RuneReader) *Parser {
	return &Parser{r: newReader(r)}
}

func (p *Parser) parseArray() (Array, error) {
	array := make(Array, 0)
	more := true
	for {
		tok, err := p.r.token()
		if err != nil {
			return nil, err
		}
		if more {
			if del, ok := tok.(tokDelim); ok && del.ch == ']' {
				break
			}
			n, err := p.parse(tok)
			if err != nil {
				return nil, err
			}
			array = append(array, n)
			more = false
		} else {
			if del, ok := tok.(tokDelim); !ok || del.ch != ',' && del.ch != ']' {
				return nil, fmt.Errorf("jtree: unexpected token at position %d: '%v'", tok.pos(), tok)
			} else if del.ch == ']' {
				break
			} else {
				more = true
			}
		}
	}
	return array, nil
}

func (p *Parser) parseObject() (*Object, error) {
	object := Object{
		keys:   make([]string, 0),
		values: make(map[string]Node),
	}
	more := true
	for {
		tok, err := p.r.token()
		if err != nil {
			return nil, err
		}
		if more {
			if del, ok := tok.(tokDelim); ok {
				if del.ch == '}' {
					break
				} else {
					return nil, fmt.Errorf("jtree: unexpected delimiter '%c' at position %d", del.ch, tok.pos())
				}
			} else {
				key, ok := tok.(tokString)
				if !ok {
					return nil, fmt.Errorf("jtree: object key expected at position %d: '%v'", tok.pos(), tok)
				}
				tok, err = p.r.token()
				if err != nil {
					return nil, err
				}
				del, ok := tok.(tokDelim)
				if !ok || del.ch != ':' {
					return nil, fmt.Errorf("jtree: colon expected at position %d: '%v'", tok.pos(), tok)
				}
				tok, err = p.r.token()
				if err != nil {
					return nil, err
				}
				value, err := p.parse(tok)
				if err != nil {
					return nil, err
				}
				object.keys = append(object.keys, key.str)
				object.values[key.str] = value
				more = false
			}
		} else {
			if del, ok := tok.(tokDelim); !ok || del.ch != ',' && del.ch != '}' {
				return nil, fmt.Errorf("jtree: unexpected token at position %d: '%v'", tok.pos(), tok)
			} else if del.ch == '}' {
				break
			} else {
				more = true
			}
		}
	}
	return &object, nil
}

func (p *Parser) parse(tok token) (Node, error) {
	switch t := tok.(type) {
	case tokString:
		return String(t.str), nil
	case tokNum:
		f, _, err := new(big.Float).Parse(t.str, 10)
		if err != nil {
			return nil, fmt.Errorf("jtree: %w", err)
		}
		return (*Num)(f), nil
	case tokDelim:
		switch t.ch {
		case '{':
			return p.parseObject()
		case '[':
			return p.parseArray()
		default:
			return nil, fmt.Errorf("jtree: unexpected delimiter '%c' at position %d", t.ch, t.p)
		}
	case tokRes:
		switch t.str {
		case "true", "false":
			return Bool(t.str == "true"), nil
		case "null":
			return Null{}, nil
		default:
			return nil, fmt.Errorf("jtree: undefined keyword '%s' at position %d", t.str, t.p)
		}
	default:
		panic("unexpected token")
	}
}

// Parse parses JSON stream into an AST representation
func (p *Parser) Parse() (Node, error) {
	tok, err := p.r.token()
	if err != nil {
		return nil, err
	}
	return p.parse(tok)
}
