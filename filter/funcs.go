package filter

import (
	"time"

	"github.com/zeebo/mwc"

	"storj.io/hydrant/value"
)

type Funcs struct{}

func SetBuiltins(p *Parser) {
	p.SetFunction("since", Funcs{}.Since)
	p.SetFunction("not", Funcs{}.Not)
	p.SetFunction("true", Funcs{}.True)
	p.SetFunction("false", Funcs{}.False)
	p.SetFunction("rand", Funcs{}.Rand)
	p.SetFunction("eq", Funcs{}.Equal)
	p.SetFunction("lt", Funcs{}.Less)
	p.SetFunction("lte", Funcs{}.LessEqual)
	p.SetFunction("gt", Funcs{}.Greater)
	p.SetFunction("gte", Funcs{}.GreaterEqual)
}

func (Funcs) Since(es *EvalState) bool {
	ts, ok := pop(es, value.Value.Timestamp)
	if !ok {
		return false
	}
	es.Push(value.Duration(time.Since(ts)))
	return true
}

func (Funcs) Not(es *EvalState) bool {
	cond, ok := pop(es, value.Value.Bool)
	if !ok {
		return false
	}
	es.Push(value.Bool(!cond))
	return true
}

func (Funcs) True(es *EvalState) bool {
	es.Push(value.Bool(true))
	return true
}

func (Funcs) False(es *EvalState) bool {
	es.Push(value.Bool(false))
	return true
}

func (Funcs) Rand(es *EvalState) bool {
	es.Push(value.Float(mwc.Float64()))
	return true
}

func (Funcs) Equal(es *EvalState) bool {
	left, right, ok := popForCompare(es)
	if !ok {
		return false
	}
	es.Push(value.Bool(value.Equal(left, right)))
	return true
}

func (Funcs) Less(es *EvalState) bool {
	left, right, ok := popForCompare(es)
	if !ok {
		return false
	}

	b, ok := valuesLess(left, right)
	es.Push(value.Bool(b && ok))
	return true
}

func (Funcs) LessEqual(es *EvalState) bool {
	left, right, ok := popForCompare(es)
	if !ok {
		return false
	}

	if b, ok := valuesLess(left, right); b && ok {
		es.Push(value.Bool(true))
		return true
	}

	es.Push(value.Bool(value.Equal(left, right)))
	return true
}

func (Funcs) Greater(es *EvalState) bool {
	left, right, ok := popForCompare(es)
	if !ok {
		return false
	}

	b, ok := valuesLess(left, right)
	es.Push(value.Bool(!b && ok))
	return true
}

func (Funcs) GreaterEqual(es *EvalState) bool {
	left, right, ok := popForCompare(es)
	if !ok {
		return false
	}

	if b, ok := valuesLess(left, right); !b && ok {
		es.Push(value.Bool(true))
		return true
	}

	es.Push(value.Bool(value.Equal(left, right)))
	return true
}

func popForCompare(es *EvalState) (left, right value.Value, ok bool) {
	right, ok = es.Pop()
	if !ok {
		return left, right, false
	}

	left, ok = es.Pop()
	if !ok {
		return left, right, false
	}

	// if they have different kinds, try to make them the same kind. note that
	// this doesn't handle a case like (float64(0.5), uint64(1<<60)) because
	// the former must be a float and the latter would lose precision as a
	// float.
	if left.Kind() != right.Kind() {
		left = upcastNumeric(left)
		right = upcastNumeric(right)
	}

	return left, right, true
}

func upcastNumeric(v value.Value) value.Value {
	if i, ok := v.Int(); ok {
		if int64(float64(i)) == i {
			v = value.Float(float64(i))
		} else if i >= 0 {
			v = value.Uint(uint64(i))
		}
	}
	if u, ok := v.Uint(); ok && uint64(float64(u)) == u {
		v = value.Float(float64(u))
	}
	return v
}

func valuesLess(left, right value.Value) (b bool, ok bool) {
	switch uint64(left.Kind())<<8 | uint64(right.Kind()) {
	default:
		return false, false

	case uint64(value.KindString)<<8 | uint64(value.KindString):
		l, _ := left.String()
		r, _ := right.String()
		return l < r, true

	case uint64(value.KindBytes)<<8 | uint64(value.KindBytes):
		l, _ := left.Bytes()
		r, _ := right.Bytes()
		return string(l) < string(r), true

	case uint64(value.KindInt)<<8 | uint64(value.KindInt):
		l, _ := left.Int()
		r, _ := right.Int()
		return l < r, true

	case uint64(value.KindUint)<<8 | uint64(value.KindUint):
		l, _ := left.Uint()
		r, _ := right.Uint()
		return l < r, true

	case uint64(value.KindDuration)<<8 | uint64(value.KindDuration):
		l, _ := left.Duration()
		r, _ := right.Duration()
		return l < r, true

	case uint64(value.KindFloat)<<8 | uint64(value.KindFloat):
		l, _ := left.Float()
		r, _ := right.Float()
		return l < r, true

	case uint64(value.KindTimestamp)<<8 | uint64(value.KindTimestamp):
		l, _ := left.Timestamp()
		r, _ := right.Timestamp()
		return l.Before(r), true
	}
}
