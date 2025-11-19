package filter

import (
	"strconv"

	"github.com/zeebo/errs/v2"
)

type Filter struct {
	parser *Parser
	filter string
	prog   []inst
	floats []float64
}

type Parser struct {
	funcs []func(*EvalState) bool
	names map[string]uint32
}

func (p *Parser) SetFunction(name string, fn func(*EvalState) bool) {
	if p.names == nil {
		p.names = make(map[string]uint32)
	}
	if n, ok := p.names[name]; !ok {
		n = uint32(len(p.funcs))
		p.names[name] = n
		p.funcs = append(p.funcs, fn)
	} else {
		p.funcs[n] = fn
	}
}

func (p *Parser) Parse(filter string) (*Filter, error) {
	toks, err := tokens(filter, nil)
	if err != nil {
		return nil, err
	}

	ps := &parseState{
		parser: p,
		toks:   toks,
		into: &Filter{
			parser: p,
			filter: filter,
		},
	}

	if err := ps.parseCompoundExpr(); err != nil {
		return nil, err
	} else if ps.tokn != uint(len(ps.toks)) {
		tok := ps.peek()
		if tok.isLiteral() {
			return nil, errs.Errorf("unexpected token: %q", tok.literal(filter))
		} else {
			return nil, errs.Errorf("unexpected token: %v", tok)
		}
	}

	return ps.into, nil
}

type parseState struct {
	parser *Parser
	toks   []token
	tokn   uint
	into   *Filter
}

func (ps *parseState) pushOp(op byte) {
	ps.pushInst(op, 0)
}

func (ps *parseState) pushInst(op byte, arg uint32) {
	ps.into.prog = append(ps.into.prog, inst{op: op, arg: arg})
}

func (ps *parseState) peek() token {
	if ps.tokn < uint(len(ps.toks)) {
		return ps.toks[ps.tokn]
	}
	return 0
}

func (ps *parseState) next() token {
	if ps.tokn < uint(len(ps.toks)) {
		s := ps.toks[ps.tokn]
		ps.tokn++
		return s
	}
	return 0
}

func (ps *parseState) nextIf(tok token) bool {
	if ps.peek() == tok {
		ps.tokn++
		return true
	}
	return false
}

func (ps *parseState) parseCompoundExpr() error {
	if err := ps.parseExpr(); err != nil {
		return err
	}

	for {
		op := ps.peekExprConjugate()
		if op == 0 {
			return nil
		}
		ps.tokn++

		if err := ps.parseExpr(); err != nil {
			return err
		}

		ps.pushOp(op)
	}
}

func (ps *parseState) peekExprConjugate() (op byte) {
	switch tok := ps.peek(); tok {
	case tokenOr:
		return instOr
	case tokenAnd:
		return instAnd
	default:
		return 0
	}
}

func (ps *parseState) parseExpr() error {
	if ps.peek() == tokenLParen {
		return ps.parseExprGroup()
	}

	tok := ps.next()
	if !tok.isLiteral() {
		return errs.Errorf("expected literal, got %v", tok)
	}

	lit := tok.literal(ps.into.filter)
	if len(lit) == 0 {
		return errs.Errorf("empty literal: %v", tok)
	}

	// if it's a function call, look up the function and parse the call body
	if !tok.isQuoted() && ps.peek() == tokenLParen {
		fn, ok := ps.parser.names[lit]
		if !ok {
			return errs.Errorf("unknown function: %q", lit)
		}

		return ps.parseCallBody(fn)
	}

	if tok.hasEscape() {
		unquoted, err := strconv.Unquote(`"` + lit + `"`)
		if err != nil {
			return errs.Errorf("invalid escape in literal %q: %w", lit, err)
		}
		tok = newLiteralToken(true, false, uint(len(ps.into.filter)), uint(len(unquoted)))
		ps.into.filter += unquoted
	}

	if float, err := strconv.ParseFloat(lit, 64); err == nil {
		ps.pushInst(instPushFloat, uint32(len(ps.into.floats)))
		ps.into.floats = append(ps.into.floats, float)
	} else {
		ps.pushInst(instPushStr, uint32(tok))
	}

	return nil
}

func (ps *parseState) parseCallBody(fn uint32) error {
	if tok := ps.next(); tok != tokenLParen {
		return errs.Errorf("expected '(', got %v", tok)
	}

	for {
		if ps.nextIf(tokenRParen) {
			break
		}

		if err := ps.parseExpr(); err != nil {
			return err
		}

		ps.nextIf(tokenComma)
	}

	ps.pushInst(instCall, fn)

	return nil
}

func (ps *parseState) parseExprGroup() error {
	if tok := ps.next(); tok != tokenLParen {
		return errs.Errorf("expected '(', got %v", tok)
	}

	if err := ps.parseCompoundExpr(); err != nil {
		return err
	}

	if tok := ps.next(); tok != tokenRParen {
		return errs.Errorf("expected ')', got %v", tok)
	}

	return nil
}
