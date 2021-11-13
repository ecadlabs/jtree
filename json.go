package jtree

import (
	"bufio"
	"bytes"
	"io"
)

// minimal encoding/json compatibility layer

func Unmarshal(data []byte, v interface{}) error {
	p := NewParser(bytes.NewReader(data))
	n, err := p.Parse()
	if err != nil {
		return err
	}
	return n.Decode(v)
}

type Decoder struct {
	p   *Parser
	opt []Option
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{p: NewParser(bufio.NewReader(r))}
}

func (dec *Decoder) Decode(v interface{}) error {
	n, err := dec.p.Parse()
	if err != nil {
		return err
	}
	return n.Decode(v, dec.opt...)
}

func (dec *Decoder) DisallowUnknownFields() {
	dec.opt = append(dec.opt, OpDisallowUnknownFields)
}
