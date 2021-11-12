package jtree

import (
	"fmt"
	"io"
	"strings"
)

type token interface {
	pos() int
	String() string
}

type tokDelim struct {
	ch int
	p  int
}

func (t tokDelim) pos() int       { return t.p }
func (t tokDelim) String() string { return string(rune(t.ch)) }

type tokString struct {
	str string
	p   int
}

func (t tokString) pos() int       { return t.p }
func (t tokString) String() string { return t.str }

type tokNum struct {
	tokString
}

type tokRes struct {
	tokString
}

func isSpace(c int) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}

func isNum(c int) bool {
	return c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.' || c == 'e' || c == 'E'
}

type reader struct {
	r       io.ByteReader
	eof     bool
	unr     int
	lastPos int
}

func newReader(r io.ByteReader) *reader {
	return &reader{r: r, unr: -1, lastPos: -1}
}

func (r *reader) byte() (b int, err error) {
	if r.unr >= 0 {
		b, r.unr, r.lastPos = r.unr, -1, r.lastPos+1
		return
	}
	c, err := r.r.ReadByte()
	if err != nil {
		if err == io.EOF {
			r.eof = true
		}
		return -1, err
	}
	b, r.lastPos = int(c), r.lastPos+1
	return
}

func (r *reader) unread(b int) {
	r.unr, r.lastPos = b, r.lastPos-1
}

func (r *reader) token() (token, error) {
	if r.eof {
		return nil, io.EOF
	}
	var (
		c   int
		err error
	)
	for ok := true; ok; ok = isSpace(c) {
		c, err = r.byte()
		if err != nil {
			return nil, err
		}
	}

	pos := r.lastPos
	switch {
	case c >= '0' && c <= '9' || c == '-' || c == '.':
		// number
		var s strings.Builder
		for {
			s.WriteByte(byte(c))
			c, err = r.byte()
			if err == io.EOF {
				break
			} else if err != nil {
				return nil, err
			} else if !isNum(c) {
				r.unread(c)
				break
			}
		}
		return tokNum{tokString{s.String(), pos}}, nil

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
			c, err = r.byte()
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
		s    strings.Builder
		esc  bool
		ln   int
		code int
	)
	code0 := -1
	for {
		c, err := r.byte()
		if err != nil {
			return "", err
		}
		if ln != 0 {
			var hex int
			switch {
			case c >= '0' && c <= '9':
				hex = c - '0'
			case c >= 'a' && c <= 'f':
				hex = c - 'a' + 0xa
			case c >= 'A' && c <= 'F':
				hex = c - 'A' + 0xa
			default:
				return "", fmt.Errorf("jtree: invalid hexadecimal digit '%c' at position %d", c, r.lastPos)
			}
			code = code<<4 | hex
			ln--
			if ln == 0 {
				if code0 >= 0 {
					if code&0xfc00 != 0xdc00 {
						return "", fmt.Errorf("jtree: invalid code %#x in surrogate pair", code)
					}
					code = code&0x03ff | (code0&0x03ff)<<10 | 0x10000
					s.WriteRune(rune(code))
					code0 = -1
				} else if code&0xfc00 == 0xd800 {
					code0 = code
				} else {
					s.WriteRune(rune(code))
				}
				code = 0
			}
		} else if esc {
			esc = false
			if c == 'u' {
				ln = 4
			} else if c == 'x' {
				ln = 2
			} else {
				code0 = -1
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
				s.WriteByte(byte(c))
			}
		} else if c == '\\' {
			esc = true
		} else {
			code0 = -1
			if c == '"' {
				break
			}
			s.WriteByte(byte(c))
		}
	}
	return s.String(), nil
}
