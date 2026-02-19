package process

import (
	"fmt"

	"storj.io/hydrant"
)

var reserved = map[string]bool{
	"file":      true,
	"func":      true,
	"line":      true,
	"message":   true,
	"timestamp": true,
	"name":      true,
	"start":     true,
	"span_id":   true,
	"parent_id": true,
	"trace_id":  true,
	"duration":  true,
	"success":   true,
	"_":         true,
}

type Store struct {
	annotations []hydrant.Annotation
}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) MustRegisterAnnotation(a ...hydrant.Annotation) {
	for i := range a {
		if reserved[a[i].Key] {
			panic(fmt.Sprintf("%q is reserved", a[i].Key))
		}
	}
	s.annotations = append(s.annotations, a...)
}

func (s *Store) Annotations() []hydrant.Annotation {
	return s.annotations
}

var (
	DefaultStore = NewStore()
)

func MustRegisterAnnotation(a ...hydrant.Annotation) {
	DefaultStore.MustRegisterAnnotation(a...)
}
