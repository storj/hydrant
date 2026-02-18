package receiver

import (
	"io"
	"net/http"

	"github.com/klauspost/compress/zstd"

	"storj.io/hydrant"
	"storj.io/hydrant/internal/rw"
)

type Handler struct {
	sub hydrant.Submitter
	dec *zstd.Decoder
}

func NewHTTPHandler(sub hydrant.Submitter) *Handler {
	dec, err := zstd.NewReader(nil,
		zstd.WithDecoderMaxMemory(64<<20),
	)
	if err != nil {
		panic(err) // this can only happen with invalid options
	}

	return &Handler{
		sub: sub,
		dec: dec,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := func() error {
		// TODO: limit size
		buf, err := io.ReadAll(req.Body)
		if err != nil {
			return err
		}

		buf, err = h.dec.DecodeAll(buf, nil)
		if err != nil {
			return err
		}

		var process hydrant.Event
		buf, err = process.ReadFrom(buf)
		if err != nil {
			return err
		}

		r := rw.NewReader(buf)
		count := r.ReadVarint()
		buf, err = r.Done()
		if err != nil {
			return err
		}

		for range count {
			var ev hydrant.Event
			buf, err = ev.ReadFrom(buf)
			if err != nil {
				return err
			}

			h.sub.Submit(req.Context(), append(process, ev...))
		}

		return nil
	}(); err != nil {
		http.Error(w, "internal service error", http.StatusInternalServerError)
		return
	}
}
