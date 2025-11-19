package filter

import (
	"storj.io/hydrant"
	"storj.io/hydrant/value"
)

type EvalState struct {
	ev    hydrant.Event
	stack []value.Value
}

func (es *EvalState) Evaluate(f *Filter, ev hydrant.Event) bool {
	es.ev = ev
	es.stack = es.stack[:0]

	for _, i := range f.prog {
		switch i.op {
		case instPushStr:
			es.Push(value.String(token(i.arg).literal(f.filter)))

		case instPushFloat:
			if int(i.arg) >= len(f.floats) {
				return false
			}
			es.Push(value.Float(f.floats[i.arg]))

		case instPushDur:
			if int(i.arg) >= len(f.durs) {
				return false
			}
			es.Push(value.Duration(f.durs[i.arg]))

		case instCall:
			if int(i.arg) >= len(f.parser.funcs) {
				return false
			}
			if !f.parser.funcs[i.arg](es) {
				return false
			}

		case instAnd:
			left, lok := pop(es, value.Value.Bool)
			right, rok := pop(es, value.Value.Bool)
			if !lok || !rok {
				return false
			}
			es.Push(value.Bool(left && right))

		case instOr:
			left, lok := pop(es, value.Value.Bool)
			right, rok := pop(es, value.Value.Bool)
			if !lok || !rok {
				return false
			}
			es.Push(value.Bool(left || right))

		case instNop:
		default:
			return false
		}
	}

	res, ok := pop(es, value.Value.Bool)
	return res && ok
}

func pop[T any](es *EvalState, convert func(value.Value) (T, bool)) (t T, ok bool) {
	if n := len(es.stack); n > 0 {
		v := es.stack[n-1]
		es.stack = es.stack[:n-1]
		t, ok = convert(v)
	}
	return t, ok
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

func (es *EvalState) Lookup(key string) (value.Value, bool) {
	// TODO: hey jt this is where i was like "man they should really be maps"
	for i := len(es.ev.System) - 1; i >= 0; i-- {
		if ann := es.ev.System[i]; ann.Key == key {
			return ann.Value, true
		}
	}
	for i := len(es.ev.User) - 1; i >= 0; i-- {
		if ann := es.ev.User[i]; ann.Key == key {
			return ann.Value, true
		}
	}
	return value.Value{}, false
}
