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
		tokDelim{',', 51},
		tokString{"num", 52},
		tokDelim{':', 57},
		tokNum{tokString{"-0.123e-5", 58}},
		tokDelim{',', 67},
		tokString{"bool", 68},
		tokDelim{':', 74},
		tokRes{tokString{"false", 75}},
		tokDelim{'}', 80},
	}, tokens)
}
