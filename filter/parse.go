package filter

import (
	"fmt"
	"strconv"
	"time"

	"github.com/zeebo/errs/v2"

	"storj.io/hydrant/value"
)

type Filter struct {
	env    *Environment
	filter string
	prog   []inst
	vals   []value.Value
}

func (f *Filter) String() string {
	return fmt.Sprintf("(filter %q %v %v)", f.filter, f.prog, anyfy(f.vals))
}

type Environment struct {
	funcs []func(*EvalState) bool
	names map[string]uint32
}

func (env *Environment) SetFunction(name string, fn func(*EvalState) bool) {
	if env.names == nil {
		env.names = map[string]uint32{"key": 0, "has": 1}
		env.funcs = append(env.funcs, intrinsicKey, intrinsicHas)
	}
	if name == "key" || name == "has" {
		panic("cannot override builtin function: " + name)
	}
	if n, ok := env.names[name]; !ok {
		n = uint32(len(env.funcs))
		env.names[name] = n
		env.funcs = append(env.funcs, fn)
	} else {
		env.funcs[n] = fn
	}
}

func intrinsicKey(es *EvalState) bool {
	key, ok := pop(es, value.Value.String)
	if !ok {
		return false
	}
	v, ok := es.Lookup(key)
	if !ok {
		return false
	}
	es.Push(v)
	return true
}

func intrinsicHas(es *EvalState) bool {
	key, ok := pop(es, value.Value.String)
	if !ok {
		return false
	}
	_, ok = es.Lookup(key)
	es.Push(value.Bool(ok))
	return true
}

func (env *Environment) Parse(filter string) (*Filter, error) {
	toks, err := tokens(filter, nil)
	if err != nil {
		return nil, err
	}

	ps := &parseState{
		parser: env,
		toks:   toks,
		into: &Filter{
			env:    env,
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

	ps.into.prog = optimize(ps.into.prog)

	return ps.into, nil
}

func optimize(prog []inst) []inst {
	// peephole optimize key/has calls on string literals. they look like
	// 	(pushStr X) (call 0) => (key X) (nop)
	// 	(pushStr X) (call 1) => (has X) (nop)
	// we insert the nop to keep instruction offsets the same for jumps.
	for i := 0; i < len(prog)-1; i++ {
		if prog[i].op == instPushStr && prog[i+1].op == instCall && prog[i+1].arg <= 1 {
			if prog[i+1].arg == 0 {
				prog[i] = inst{op: instKey, arg: prog[i].arg}
			} else {
				prog[i] = inst{op: instHas, arg: prog[i].arg}
			}
			prog[i+1] = inst{op: instNop}
		}
	}

	// optimize a jumpFalse/True that targets a jumpFalse/True to target what the next one targets
	// instead. we do this backwards to avoid O(n^2) behavior because future jumps will already be
	// set to their final targets. this way we also don't need to loop to a fixed point.
	for i := len(prog) - 1; i >= 0; i-- {
		if prog[i].op != instJumpFalse && prog[i].op != instJumpTrue {
			continue
		}

		target := int(i) + int(prog[i].arg) + 1
		if target < 0 || target >= len(prog) {
			continue
		}

		// if the next op will do the same thing, jump to where it jumps to instead. because it is
		// later in the instruction stream, it is already optimally targeted.
		if prog[target].op == prog[i].op {
			prog[i].arg += prog[target].arg + 1
		}
	}

	// we're going to remove nops. we do this in two steps. first, before we remove them, we fix up
	// any jumps to account for the nops that will be removed. then we remove the nops. we do this
	// so that indexes stay consistent while we fix up jumps.
	for i := range prog {
		if prog[i].op != instJumpFalse && prog[i].op != instJumpTrue {
			continue
		}
		for _, inst := range prog[i+1 : i+int(prog[i].arg)] {
			if inst.op == instNop {
				prog[i].arg--
			}
		}
	}

	// remove the nops now that all the jumps are expecting them to be gone.
	out := prog[:0]
	for _, inst := range prog {
		if inst.op != instNop {
			out = append(out, inst)
		}
	}
	prog = out

	return prog
}

type parseState struct {
	parser *Environment
	toks   []token
	tokn   uint
	into   *Filter
}

func (ps *parseState) pushOp(op byte) int {
	return ps.pushInst(op, 0)
}

func (ps *parseState) pushInst(op byte, arg uint32) int {
	ps.into.prog = append(ps.into.prog, inst{op: op, arg: arg})
	return len(ps.into.prog) - 1
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

		// insert placeholder for jump
		n := ps.pushOp(instNop)

		if err := ps.parseExpr(); err != nil {
			return err
		}

		// fixup jump now that we know how many instructions to advance
		switch offset := uint32(len(ps.into.prog) - n); op {
		case instAnd:
			ps.into.prog[n] = inst{op: instJumpFalse, arg: offset}
		case instOr:
			ps.into.prog[n] = inst{op: instJumpTrue, arg: offset}
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

	// if it has an escape, unquote it and update the token to point to the unescaped literal that
	// we append to the filter lol.
	if tok.hasEscape() {
		unquoted, err := strconv.Unquote(`"` + lit + `"`)
		if err != nil {
			return errs.Errorf("invalid escape in literal %q: %w", lit, err)
		}
		tok = newLiteralToken(true, false, uint(len(ps.into.filter)), uint(len(unquoted)))
		ps.into.filter += unquoted

		if tok.literal(ps.into.filter) != unquoted {
			return errs.Errorf("internal error: token literal mismatch after unescape")
		}
		lit = unquoted
	}

	var v value.Value
	if in, err := strconv.ParseInt(lit, 0, 64); err == nil {
		v = value.Int(in)
	} else if un, err := strconv.ParseUint(lit, 0, 64); err == nil {
		v = value.Uint(un)
	} else if dur, err := time.ParseDuration(lit); err == nil {
		v = value.Duration(dur)
	} else if float, err := strconv.ParseFloat(lit, 64); err == nil {
		v = value.Float(float)
	} else {
		ps.pushInst(instPushStr, uint32(tok))
		return nil
	}

	ps.pushInst(instPushVal, uint32(len(ps.into.vals)))
	ps.into.vals = append(ps.into.vals, v)
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
