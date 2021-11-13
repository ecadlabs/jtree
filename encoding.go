package jtree

import (
	"encoding/base64"
	"encoding/hex"
)

// Encoding is an interface used for string encoded binary values
type Encoding interface {
	Encode([]byte) []byte
	Decode([]byte) ([]byte, error)
}

type base64Encoding struct{}

func (base64Encoding) Encode(src []byte) []byte {
	buf := make([]byte, base64.StdEncoding.EncodedLen(len(src)))
	base64.StdEncoding.Encode(buf, src)
	return buf
}

func (base64Encoding) Decode(src []byte) ([]byte, error) {
	buf := make([]byte, base64.StdEncoding.DecodedLen(len(src)))
	n, err := base64.StdEncoding.Decode(buf, src)
	return buf[:n], err
}

type hexEncoding struct{}

func (hexEncoding) Encode(src []byte) []byte {
	buf := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(buf, src)
	return buf
}

func (hexEncoding) Decode(src []byte) ([]byte, error) {
	buf := make([]byte, hex.DecodedLen(len(src)))
	n, err := hex.Decode(buf, src)
	return buf[:n], err
}

var (
	// Base64 is the standard base64 encoding
	Base64 Encoding = base64Encoding{}
	// Hex is the hex encoding (([0-9a-fA-F]{2})*)
	Hex Encoding = hexEncoding{}
)
