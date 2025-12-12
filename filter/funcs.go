package filter

import (
	"time"

	"github.com/zeebo/mwc"

	"storj.io/hydrant/value"
)

type Funcs struct{}

func SetBuiltins(env *Environment) {
	env.SetFunction("since", Funcs{}.Since)
	env.SetFunction("not", Funcs{}.Not)
	env.SetFunction("true", Funcs{}.True)
	env.SetFunction("false", Funcs{}.False)
	env.SetFunction("rand", Funcs{}.Rand)
	env.SetFunction("eq", Funcs{}.Equal)
	env.SetFunction("lt", Funcs{}.Less)
	env.SetFunction("lte", Funcs{}.LessEqual)
	env.SetFunction("gt", Funcs{}.Greater)
	env.SetFunction("gte", Funcs{}.GreaterEqual)
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

	b, ok := value.Less(left, right)
	es.Push(value.Bool(b && ok))
	return true
}

func (Funcs) LessEqual(es *EvalState) bool {
	left, right, ok := popForCompare(es)
	if !ok {
		return false
	}

	if b, ok := value.Less(left, right); b && ok {
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

	b, ok := value.Less(left, right)
	es.Push(value.Bool(!b && ok))
	return true
}

func (Funcs) GreaterEqual(es *EvalState) bool {
	left, right, ok := popForCompare(es)
	if !ok {
		return false
	}

	if b, ok := value.Less(left, right); !b && ok {
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
