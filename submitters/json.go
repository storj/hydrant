package submitters

import (
	"strconv"

	"storj.io/hydrant"
)

type jsonEvent []jsonAnnotation

type jsonAnnotation struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func serializeEvents(events []hydrant.Event) []jsonEvent {
	out := make([]jsonEvent, len(events))
	for i, ev := range events {
		out[i] = serializeEvent(ev)
	}
	return out
}

func serializeEvent(ev hydrant.Event) jsonEvent {
	out := make(jsonEvent, len(ev))
	for i, a := range ev {
		// TODO: using String is wrong. this should do its own serialization.
		v := a.String()[len(a.Key)+1:] // strip "key=" prefix from String()
		if t, ok := a.Value.Timestamp(); ok {
			v = strconv.FormatInt(t.UnixNano(), 10)
		}
		out[i] = jsonAnnotation{
			Key:   a.Key,
			Value: v,
		}
	}
	return out
}
