package filter

import (
	"github.com/zeebo/mwc"

	"storj.io/hydrant/value"
)

type Funcs struct{}

func SetBuiltins(p *Parser) {
	p.SetFunction("equal", Funcs{}.Equal)
	p.SetFunction("exists", Funcs{}.Exists)
	p.SetFunction("key", Funcs{}.Key)
	p.SetFunction("less", Funcs{}.Less)
	p.SetFunction("true", func(es *EvalState) bool { es.Push(value.Bool(true)); return true })
	p.SetFunction("rand", func(es *EvalState) bool { es.Push(value.Float(mwc.Float64())); return true })
}

func (Funcs) Equal(es *EvalState) bool {
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

func (Funcs) Exists(es *EvalState) bool {
	key, ok := pop(es, value.Value.String)
	if !ok {
		return false
	}
	_, ok = es.Lookup(key)
	es.Push(value.Bool(ok))
	return true
}

func (Funcs) Key(es *EvalState) bool {
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

func (Funcs) Less(es *EvalState) bool {
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
