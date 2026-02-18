package rw

import (
	"encoding/binary"

	"github.com/zeebo/errs/v2"

	"github.com/histdb/histdb/buffer"
	"github.com/histdb/histdb/varint"
)

type Reader struct {
	buf []byte
	err error
}

func NewReader(buf []byte) *Reader {
	return &Reader{buf: buf}
}

func (r *Reader) Invalid(err error) {
	if r.err == nil {
		r.err = errs.Wrap(err)
		r.buf = nil
	}
}

func (r *Reader) ReadVarint() (x uint64) {
	if len(r.buf) >= 9 {
		var ndec uintptr
		ndec, x = varint.FastConsume((*[9]byte)(r.buf[:]))
		r.buf = r.buf[ndec:]
	} else {
		var rem buffer.T
		var ok bool
		x, rem, ok = varint.Consume(buffer.OfLen(r.buf))
		if !ok {
			r.Invalid(errs.Errorf("short buffer"))
		} else {
			r.buf = rem.Suffix()
		}
	}
	return
}

func (r *Reader) ReadUint8() (x uint8) {
	if len(r.buf) >= 1 {
		x = r.buf[0]
		r.buf = r.buf[1:]
	} else {
		r.Invalid(errs.Errorf("short buffer"))
	}
	return
}

func (r *Reader) ReadUint64() (x uint64) {
	if len(r.buf) >= 8 {
		x = binary.LittleEndian.Uint64(r.buf[:8])
		r.buf = r.buf[8:]
	} else {
		r.Invalid(errs.Errorf("short buffer"))
	}
	return x
}

func (r *Reader) ReadBytes(n uint64) (x []byte) {
	if uint64(len(r.buf)) >= n {
		x = r.buf[:n]
		r.buf = r.buf[n:]
	} else {
		r.Invalid(errs.Errorf("short buffer"))
	}
	return x
}

func (r *Reader) Done() ([]byte, error) {
	buf := r.buf
	r.buf = nil
	return buf, r.err
}
