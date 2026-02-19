package submitters

import (
	"encoding/json"
	"fmt"
	"net/http"

	"storj.io/hydrant"
	"storj.io/hydrant/internal/utils"
)

const liveBufferSize = 128

type liveBuffer struct {
	buf *utils.RingBuffer[hydrant.Event]
}

func newLiveBuffer() liveBuffer {
	return liveBuffer{buf: utils.NewRingBuffer[hydrant.Event](liveBufferSize)}
}

func (l *liveBuffer) Record(ev hydrant.Event) {
	l.buf.Add(ev)
}

func (l *liveBuffer) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("watch") != "" {
			l.handleWatch(w, r)
			return
		}
		l.handleSnapshot(w, r)
	})
}

func (l *liveBuffer) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	events := l.buf.Get()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(serializeEvents(events))
}

func (l *liveBuffer) handleWatch(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush()

	l.buf.Watch(r.Context(), func(ev hydrant.Event) {
		data, err := json.Marshal(serializeEvent(ev))
		if err != nil {
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	})
}
