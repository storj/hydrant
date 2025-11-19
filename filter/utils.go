package filter

import "storj.io/hydrant/value"

func pop[T any](es *EvalState, convert func(value.Value) (T, bool)) (t T, ok bool) {
	if n := len(es.stack); n > 0 {
		v := es.stack[n-1]
		es.stack = es.stack[:n-1]
		t, ok = convert(v)
	}
	return t, ok
}

func peek[T any](es *EvalState, convert func(value.Value) (T, bool)) (t T, ok bool) {
	if n := len(es.stack); n > 0 {
		t, ok = convert(es.stack[n-1])
	}
	return t, ok
}

func anyfy(vs []value.Value) []any {
	av := make([]any, len(vs))
	for i, v := range vs {
		av[i] = v.AsAny()
	}
	return av
}
