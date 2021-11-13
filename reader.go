package jtree

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf16"
)

type token interface {
	pos() int64
	String() string
}

type tokDelim struct {
	ch rune
	p  int64
}

func (t tokDelim) pos() int64     { return t.p }
func (t tokDelim) String() string { return string(rune(t.ch)) }

type tokString struct {
	str string
	p   int64
}

func (t tokString) pos() int64     { return t.p }
func (t tokString) String() string { return t.str }

type tokNum struct {
	tokString
}

type tokRes struct {
	tokString
}

func isSpace(c rune) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}

func isNum(c rune) bool {
	return c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.' || c == 'e' || c == 'E'
}

type reader struct {
	r   io.RuneReader
	eof bool
	unr int
	off int64
}

func newReader(r io.RuneReader) *reader {
	return &reader{r: r, unr: -1}
}

func (r *reader) pos() int64 { return r.off - 1 }

func (r *reader) rune() (v rune, err error) {
	if r.unr >= 0 {
		v, r.unr, r.off = rune(r.unr), -1, r.off+1
		return
	}
	c, _, err := r.r.ReadRune()
	if err != nil {
		if err == io.EOF {
			r.eof = true
		}
		return 0, err
	}
	v, r.off = c, r.off+1
	return
}

func (r *reader) unread(b rune) {
	r.unr, r.off = int(b), r.off-1
}

func (r *reader) token() (token, error) {
	if r.eof {
		return nil, io.EOF
	}
	var (
		c   rune
		err error
	)
	for ok := true; ok; ok = isSpace(c) {
		c, err = r.rune()
		if err != nil {
			return nil, err
		}
	}

	pos := r.pos()
	switch {
	case c >= '0' && c <= '9' || c == '-' || c == '.':
		// number
		s := make([]rune, 0)
		for {
			s = append(s, c)
			c, err = r.rune()
			if err == io.EOF {
				break
			} else if err != nil {
				return nil, err
			} else if !isNum(c) {
				r.unread(c)
				break
			}
		}
		return tokNum{tokString{string(s), pos}}, nil

	case c == '"':
		s, err := r.string()
		if err != nil {
			return nil, err
		}
		return tokString{s, pos}, err

	case c == '{' || c == '}' || c == '[' || c == ']' || c == ',' || c == ':':
		return tokDelim{c, pos}, nil

	case c >= 'a' && c <= 'z':
		// keyword
		var s strings.Builder
		for {
			s.WriteByte(byte(c))
			c, err = r.rune()
			if err == io.EOF {
				break
			} else if err != nil {
				return nil, err
			} else if !(c >= 'a' && c <= 'z') {
				r.unread(c)
				break
			}
		}
		return tokRes{tokString{s.String(), pos}}, nil

	default:
		return nil, fmt.Errorf("jtree: unexpected character '%c' at position %d", c, pos)
	}
}

func (r *reader) string() (string, error) {
	var (
		esc  bool
		ln   int
		code uint
	)
	u16 := make([]uint16, 0)
	for {
		c, err := r.rune()
		if err != nil {
			return "", err
		}
		if ln != 0 {
			var hex uint
			switch {
			case c >= '0' && c <= '9':
				hex = uint(c) - '0'
			case c >= 'a' && c <= 'f':
				hex = uint(c) - 'a' + 0xa
			case c >= 'A' && c <= 'F':
				hex = uint(c) - 'A' + 0xa
			default:
				return "", fmt.Errorf("jtree: invalid hexadecimal digit '%c' at position %d", c, r.pos())
			}
			code = code<<4 | hex
			ln--
			if ln == 0 {
				u16 = append(u16, uint16(code))
				code = 0
			}
		} else if esc {
			esc = false
			if c == 'u' {
				ln = 4
			} else if c == 'x' {
				ln = 2
			} else {
				switch c {
				case 'b':
					c = '\b'
				case 'f':
					c = '\f'
				case 'n':
					c = '\n'
				case 'r':
					c = '\r'
				case 't':
					c = '\t'
				}
				u16 = append(u16, utf16.Encode([]rune{c})...)
			}
		} else if c == '\\' {
			esc = true
		} else {
			if c == '"' {
				break
			}
			u16 = append(u16, utf16.Encode([]rune{c})...)
		}
	}
	return string(utf16.Decode(u16)), nil
}
