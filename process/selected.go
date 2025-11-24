package process

import (
	"storj.io/hydrant"
	"storj.io/hydrant/config"
)

type Selected struct {
	annotations []hydrant.Annotation
}

func NewSelected(s *Store, fields []config.Expression) *Selected {
	selected := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		selected[string(f)] = struct{}{}
	}

	rv := &Selected{}
	for _, a := range s.Annotations() {
		if _, exists := selected[a.Key]; exists {
			rv.annotations = append(rv.annotations, a)
		}
	}
	return rv
}

func (s *Selected) Annotations() []hydrant.Annotation {
	return s.annotations
}
