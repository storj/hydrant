package process

import (
	"fmt"

	"storj.io/hydrant"
)

type Store struct {
	reserved    map[string]bool
	annotations []hydrant.Annotation
}

func NewStore() *Store {
	return &Store{
		reserved: map[string]bool{
			"file":      true,
			"func":      true,
			"line":      true,
			"message":   true,
			"timestamp": true,
			"name":      true,
			"start":     true,
			"span_id":   true,
			"parent_id": true,
			"task_id":   true,
			"duration":  true,
			"success":   true,
		}}
}

func (s *Store) MustRegisterAnnotation(a ...hydrant.Annotation) {
	for i := range a {
		if s.reserved[a[i].Key] {
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
