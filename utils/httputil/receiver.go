package httputil

import (
	"io"
	"net/http"

	"github.com/klauspost/compress/zstd"

	"storj.io/hydrant"
	"storj.io/hydrant/internal/rw"
)

func NewReceiver(sub hydrant.Submitter) http.Handler {
	dec, err := zstd.NewReader(nil,
		zstd.WithDecoderMaxMemory(64<<20),
	)
	if err != nil {
		panic(err) // this can only happen with invalid options
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
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

			buf, err = dec.DecodeAll(buf, nil)
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

				sub.Submit(req.Context(), append(process, ev...))
			}

			return nil
		}(); err != nil {
			http.Error(w, "internal service error", http.StatusInternalServerError)
			return
		}
	})
}
