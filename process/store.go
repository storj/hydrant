package process

import (
	"fmt"

	"storj.io/hydrant"
	"storj.io/hydrant/value"
)

type AnnotationThunk struct {
	Key   string
	Value func() (_ value.Value, ok bool)
}

type Store struct {
	reserved         map[string]bool
	annotations      []hydrant.Annotation
	annotationThunks []AnnotationThunk
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

func (s *Store) MustRegisterAnnotationThunk(a ...AnnotationThunk) {
	for i := range a {
		if s.reserved[a[i].Key] {
			panic(fmt.Sprintf("%q is reserved", a[i].Key))
		}
	}
	s.annotationThunks = append(s.annotationThunks, a...)
}

func (s *Store) Annotations() []hydrant.Annotation {
	rv := make([]hydrant.Annotation, 0, len(s.annotations)+len(s.annotationThunks))
	rv = append(rv, s.annotations...)
	for _, a := range s.annotationThunks {
		if v, ok := a.Value(); ok {
			rv = append(rv, hydrant.Annotation{
				Key:   a.Key,
				Value: v,
			})
		}
	}
	return rv
}

var (
	DefaultStore = NewStore()
)

func MustRegisterAnnotation(a ...hydrant.Annotation) {
	DefaultStore.MustRegisterAnnotation(a...)
}

func MustRegisterAnnotationThunk(a ...AnnotationThunk) {
	DefaultStore.MustRegisterAnnotationThunk(a...)
}
