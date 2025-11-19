package filter

import (
	"fmt"

	"github.com/zeebo/errs/v2"
)

// non-literal tokens are always of the form 0b00000000_00000000_0XXXXXXX_0YYYYYYY because they
// consist of up to two ascii characters.
//
// literals are of the form 0bFBBBBBBB_BBBBBBBB_FLLLLLLL_LLLLLLLL where L and B are the length and
// byte offset of the token the two F bits mean:
//
//	0b00 : not a literal
//	0b01 : is an unquoted literal, no escapes
//	0b10 : is a quoted literal, no escapes
//	0b11 : is a quoted literal, has escapes
type token uint32

const (
	tokenInvalid token = 0

	tokenOr  token = '|'
	tokenAnd token = '&'

	tokenLParen token = '('
	tokenRParen token = ')'
	tokenComma  token = ','
)

func newLiteralToken(quoted, escaped bool, pos, length uint) (t token) {
	if quoted {
		t |= 1 << 31
		if escaped {
			t |= 1 << 15
		}
	} else {
		t |= 1 << 15
	}
	t |= token((pos & 0x7FFF) << 16)
	t |= token(length & 0x7FFF)
	return t
}

func (t token) isLiteral() bool { return t&(1<<31|1<<15) != 0 }
func (t token) isQuoted() bool  { return t&(1<<31) != 0 }
func (t token) hasEscape() bool { return t&(1<<31|1<<15) == 1<<31|1<<15 }

func (t token) litBounds() (uint, uint) {
	l, b := uint(t&0x7FFF), uint(t>>16&0x7FFF)
	return b, b + l
}

func (t token) literal(x string) string {
	if b, e := t.litBounds(); b < e && e <= uint(len(x)) {
		return x[b:e]
	}
	return ""
}

func (t token) String() string {
	if t.isLiteral() {
		b, e := t.litBounds()
		return fmt.Sprintf("lit[%d:%d]", b, e)
	}
	if t == tokenInvalid {
		return "invalid"
	}
	if t >= 256 {
		return fmt.Sprintf("%c%c", t>>8, t&0xFF)
	}
	return fmt.Sprintf("%c", t)
}

func tokens(x string, into []token) ([]token, error) {
	if uint(len(x)) > 1<<15 {
		return nil, errs.Errorf("query too long")
	}
	for pos := uint(0); uint(pos) < uint(len(x)); {
		t, n := nextToken(pos, x)
		if n == 0 {
			return nil, errs.Errorf("invalid token: %q", x[pos:])
		} else if t == 0 {
			break
		}
		into = append(into, t)
		pos += n
	}
	return into, nil
}

func nextToken(pos uint, x string) (t token, l uint) {
	if pos >= uint(len(x)) {
		return tokenInvalid, 0
	}
	x = x[pos:]

	for len(x) > 0 && (x[0] == ' ' || x[0] == '\t') {
		x = x[1:]
		pos++
		l++
	}

	// nothing left
	if len(x) == 0 {
		return tokenInvalid, l
	}

	// quoted strings
	if c := x[0]; c == '"' || c == '\'' {
		escape := false
		for i := uint(1); i < uint(len(x)); i++ {
			if x[i] == '\\' {
				if i+1 >= uint(len(x)) {
					return tokenInvalid, 0
				}
				escape = true
				i++
				continue
			}
			if x[i] == c {
				return newLiteralToken(true, escape, pos+1, i-1), l + i + 1
			}
		}
		return tokenInvalid, 0
	}

	// length 2 operators
	if len(x) > 1 {
		switch u := uint16(x[0])<<8 | uint16(x[1]); u {
		case
			'&'<<8 | '&', '|'<<8 | '|': // conjunctives
			return token(x[0]), l + 2
		}
	}

	// length 1 operators
	switch x[0] {
	case
		'(', ')', // function call, grouping
		'&', '|', // conjunctives
		',': /**/ // parameter separator
		return token(x[0]), l + 1
	}

	// strings of literal characters
	for i := range uint(len(x)) {
		c := x[i]

		if c == ' ' || c == '\t' || c == '(' || c == ')' || c == ',' {
			return newLiteralToken(false, false, pos, i), l + i
		}
	}

	return newLiteralToken(false, false, pos, uint(len(x))), l + uint(len(x))
}
