package filter

import (
	"storj.io/hydrant"
	"storj.io/hydrant/value"
)

type EvalState struct {
	ev       hydrant.Event
	stack    []value.Value
	executed int
}

func (es *EvalState) Evaluate(f *Filter, ev hydrant.Event) bool {
	es.ev = ev
	es.stack = es.stack[:0]
	es.executed = 0

	for pc := uint(0); pc < uint(len(f.prog)); pc++ {
		i := f.prog[pc]
		es.executed++

		switch i.op {
		case instNop:

		case instPushStr:
			es.Push(value.String(token(i.arg).literal(f.filter)))

		case instPushVal:
			if int(i.arg) >= len(f.vals) {
				return false
			}
			es.Push(f.vals[i.arg])

		case instCall:
			if int(i.arg) >= len(f.parser.funcs) {
				return false
			}
			if !f.parser.funcs[i.arg](es) {
				return false
			}

		case instKey:
			ev, ok := es.Lookup(token(i.arg).literal(f.filter))
			if !ok {
				return false
			}
			es.Push(ev)

		case instHas:
			_, ok := es.Lookup(token(i.arg).literal(f.filter))
			es.Push(value.Bool(ok))

		case instAnd:
			right, rok := pop(es, value.Value.Bool)
			left, lok := pop(es, value.Value.Bool)
			if !lok || !rok {
				return false
			}
			es.Push(value.Bool(left && right))

		case instJumpFalse:
			cond, ok := peek(es, value.Value.Bool)
			if !ok {
				return false
			}
			if !cond {
				pc += uint(i.arg)
			}

		case instOr:
			right, rok := pop(es, value.Value.Bool)
			left, lok := pop(es, value.Value.Bool)
			if !lok || !rok {
				return false
			}
			es.Push(value.Bool(left || right))

		case instJumpTrue:
			cond, ok := peek(es, value.Value.Bool)
			if !ok {
				return false
			}
			if cond {
				pc += uint(i.arg)
			}

		default:
			return false
		}
	}

	res, ok := pop(es, value.Value.Bool)
	return res && ok
}

func (es *EvalState) Push(v value.Value) {
	es.stack = append(es.stack, v)
}

func (es *EvalState) Pop() (value.Value, bool) {
	if n := len(es.stack); n > 0 {
		v := es.stack[n-1]
		es.stack = es.stack[:n-1]
		return v, true
	}
	return value.Value{}, false
}

func (es *EvalState) Peek() (value.Value, bool) {
	if n := len(es.stack); n > 0 {
		return es.stack[n-1], true
	}
	return value.Value{}, false
}

func (es *EvalState) Lookup(key string) (value.Value, bool) {
	s := es.ev
	for i := len(s) - 1; i >= 0; i-- {
		if ann := s[i]; ann.Key == key {
			return ann.Value, true
		}
	}
	return value.Value{}, false
}
