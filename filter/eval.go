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

func (es *EvalState) Equal() bool {
	leftVal, ok := es.Pop()
	if !ok {
		return false
	}
	rightVal, ok := es.Pop()
	if !ok {
		return false
	}
	es.Push(value.Bool(value.Equal(leftVal, rightVal)))
	return true
}

func (es *EvalState) Exists() bool {
	key, ok := pop(es, value.Value.String)
	if !ok {
		return false
	}
	_, ok = es.Lookup(key)
	es.Push(value.Bool(ok))
	return true
}

func (es *EvalState) Key() bool {
	key, ok := pop(es, value.Value.String)
	if !ok {
		return false
	}
	ev, ok := es.Lookup(key)
	if !ok {
		return false
	}
	es.Push(ev)
	return true
}

func (es *EvalState) Less() bool {
	rightVal, ok := es.Pop()
	if !ok {
		return false
	}

	leftVal, ok := es.Pop()
	if !ok {
		return false
	}

	var b bool
	switch uint64(leftVal.Kind())<<8 | uint64(rightVal.Kind()) {
	default:
		return false

	case uint64(value.KindString)<<8 | uint64(value.KindString):
		l, _ := leftVal.String()
		r, _ := rightVal.String()
		b = l < r

	case uint64(value.KindBytes)<<8 | uint64(value.KindBytes):
		l, _ := leftVal.Bytes()
		r, _ := rightVal.Bytes()
		b = string(l) < string(r)

	case uint64(value.KindInt)<<8 | uint64(value.KindInt):
		l, _ := leftVal.Int()
		r, _ := rightVal.Int()
		b = l < r
	case uint64(value.KindUint)<<8 | uint64(value.KindUint):
		l, _ := leftVal.Uint()
		r, _ := rightVal.Uint()
		b = l < r

	case uint64(value.KindDuration)<<8 | uint64(value.KindDuration):
		l, _ := leftVal.Duration()
		r, _ := rightVal.Duration()
		b = l < r

	case uint64(value.KindFloat)<<8 | uint64(value.KindFloat):
		l, _ := leftVal.Float()
		r, _ := rightVal.Float()
		b = l < r

	case uint64(value.KindTimestamp)<<8 | uint64(value.KindTimestamp):
		l, _ := leftVal.Timestamp()
		r, _ := rightVal.Timestamp()
		b = l.Before(r)
	}

	es.Push(value.Bool(b))
	return true
}
