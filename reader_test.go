package jtree

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReader(t *testing.T) {
	src := `{"str":"\\z\zz\t\n\"xxx\uD834\uDD1Efff\u1234привет","num":-0.123e-5,"bool":false}`
	r := newReader(strings.NewReader(src))
	var tokens []token
	for {
		tok, err := r.token()
		if err == io.EOF {
			break
		}
		tokens = append(tokens, tok)
	}
	require.Equal(t, []token{
		tokDelim{'{', 0},
		tokString{"str", 1},
		tokDelim{':', 6},
		tokString{"\\zzz\t\n\"xxx\U0001D11Efff\u1234привет", 7},
		tokDelim{',', 57},
		tokString{"num", 58},
		tokDelim{':', 63},
		tokNum{tokString{"-0.123e-5", 64}},
		tokDelim{',', 73},
		tokString{"bool", 74},
		tokDelim{':', 80},
		tokRes{tokString{"false", 81}},
		tokDelim{'}', 86},
	}, tokens)
}
