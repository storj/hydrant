package value

import (
	"encoding/binary"

	"github.com/zeebo/errs/v2"
)

type reader struct {
	buf []byte
	err error
}

func (r *reader) Invalid(err error) {
	if r.err == nil {
		r.err = errs.Wrap(err)
		r.buf = nil
	}
}

func (r *reader) ReadUint8() (x uint8) {
	if len(r.buf) >= 1 {
		x = r.buf[0]
		r.buf = r.buf[1:]
	} else {
		r.Invalid(errs.Errorf("short buffer"))
	}
	return
}

func (r *reader) ReadUint64() (x uint64) {
	if len(r.buf) >= 8 {
		x = binary.LittleEndian.Uint64(r.buf[:8])
		r.buf = r.buf[8:]
	} else {
		r.Invalid(errs.Errorf("short buffer"))
	}
	return x
}

func (r *reader) ReadBytes(n uint64) (x []byte) {
	if uint64(len(r.buf)) >= n {
		x = r.buf[:n]
		r.buf = r.buf[n:]
	} else {
		r.Invalid(errs.Errorf("short buffer"))
	}
	return x
}

func (r *reader) Done() ([]byte, error) {
	buf := r.buf
	r.buf = nil
	return buf, r.err
}
