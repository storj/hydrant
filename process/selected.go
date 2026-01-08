package process

import (
	"storj.io/hydrant"
)

func (s *Store) Select(fields []string) []hydrant.Annotation {
	selected := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		selected[string(f)] = struct{}{}
	}

	var rv []hydrant.Annotation
	for _, a := range s.Annotations() {
		if _, exists := selected[a.Key]; exists {
			rv = append(rv, a)
		}
	}
	return rv
}
