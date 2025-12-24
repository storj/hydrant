package utils

import "iter"

func Set[T comparable](seq iter.Seq[T]) map[T]struct{} {
	m := make(map[T]struct{})
	for x := range seq {
		m[x] = struct{}{}
	}
	return m
}
