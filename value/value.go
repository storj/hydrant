package value

import (
	"bytes"
	"encoding/binary"
	"math"
	"time"
	"unsafe"

	"github.com/zeebo/errs/v2"

	"storj.io/hydrant/rw"

	"github.com/histdb/histdb/flathist"
)

var sentinels [7]byte

type Kind uint8

const (
	KindEmpty Kind = 0

	// pointer kinds

	KindString    Kind = 1
	KindBytes     Kind = 2
	KindHistogram Kind = 3
	KindTraceId   Kind = 4

	// value kinds

	KindSpanId    Kind = 5
	KindInt       Kind = 6
	KindUint      Kind = 7
	KindDuration  Kind = 8
	KindFloat     Kind = 9
	KindBool      Kind = 10
	KindTimestamp Kind = 11

	kindLargest Kind = 12
)

const (
	pointerCount   = 5
	pointerLoShift = 3
	pointerHiShift = 64 - pointerLoShift
	pointerLoMask  = 1<<pointerLoShift - 1
	pointerHiMask  = pointerLoMask << pointerHiShift
)

func (k Kind) sentinel() unsafe.Pointer {
	return unsafe.Pointer(&sentinels[k-pointerCount])
}

func isSentinel(ptr unsafe.Pointer) bool {
	return uintptr(ptr)-uintptr(unsafe.Pointer(&sentinels[0])) < uintptr(len(sentinels))
}

type Value struct {
	_ [0]func() // no equality, must use Equal/Less

	ptr  unsafe.Pointer
	data uint64
}

func (v Value) Kind() Kind {
	d := uintptr(v.ptr) - uintptr(unsafe.Pointer(&sentinels[0]))
	if d < uintptr(len(sentinels)) {
		return Kind(d + pointerCount)
	}
	return Kind(v.data >> pointerHiShift)
}

func String(x string) (v Value) {
	if uint64(len(x))>>pointerHiShift == 0 {
		v = Value{
			ptr:  unsafe.Pointer(unsafe.StringData(x)),
			data: uint64(KindString)<<pointerHiShift | uint64(len(x)),
		}
	}
	return v
}

func (v Value) String() (x string, ok bool) {
	if ok = v.data>>pointerHiShift == uint64(KindString) && !isSentinel(v.ptr); ok {
		x = unsafe.String((*byte)(v.ptr), int(v.data&^pointerHiMask))
	}
	return x, ok
}

func Bytes(x []byte) (v Value) {
	if uint64(len(x))>>pointerHiShift == 0 {
		v = Value{
			ptr:  unsafe.Pointer(unsafe.SliceData(x)),
			data: uint64(KindBytes)<<pointerHiShift | uint64(len(x)),
		}
	}
	return v
}

func (v Value) Bytes() (x []byte, ok bool) {
	if ok = v.data>>pointerHiShift == uint64(KindBytes) && !isSentinel(v.ptr); ok {
		x = unsafe.Slice((*byte)(v.ptr), int(v.data&^pointerHiMask))
	}
	return x, ok
}

func Histogram(x *flathist.Histogram) (v Value) {
	if x != nil {
		v = Value{
			ptr:  unsafe.Pointer(x),
			data: uint64(KindHistogram) << pointerHiShift,
		}
	}
	return v
}

func (v Value) Histogram() (x *flathist.Histogram, ok bool) {
	if ok = v.data>>pointerHiShift == uint64(KindHistogram) && !isSentinel(v.ptr); ok {
		x = (*flathist.Histogram)(v.ptr)
	}
	return x, ok
}

func TraceId(x [16]byte) (v Value) {
	return Value{
		ptr:  unsafe.Pointer(&x),
		data: uint64(KindTraceId) << pointerHiShift,
	}
}

func (v Value) TraceId() (x [16]byte, ok bool) {
	if ok = v.data>>pointerHiShift == uint64(KindTraceId) && !isSentinel(v.ptr); ok {
		x = *(*[16]byte)(v.ptr)
	}
	return x, ok
}

func Int(x int64) Value {
	return Value{
		ptr:  KindInt.sentinel(),
		data: uint64(x),
	}
}

func (v Value) Int() (x int64, ok bool) {
	if ok = v.ptr == KindInt.sentinel(); ok {
		x = int64(v.data)
	}
	return x, ok
}

func Uint(x uint64) Value {
	return Value{
		ptr:  KindUint.sentinel(),
		data: x,
	}
}

func (v Value) Uint() (x uint64, ok bool) {
	if ok = v.ptr == KindUint.sentinel(); ok {
		x = v.data
	}
	return x, ok
}

func Duration(x time.Duration) Value {
	return Value{
		ptr:  KindDuration.sentinel(),
		data: uint64(x),
	}
}

func (v Value) Duration() (x time.Duration, ok bool) {
	if ok = v.ptr == KindDuration.sentinel(); ok {
		x = time.Duration(v.data)
	}
	return x, ok
}

func Float(x float64) Value {
	return Value{
		ptr:  KindFloat.sentinel(),
		data: math.Float64bits(x),
	}
}

func (v Value) Float() (x float64, ok bool) {
	if ok = v.ptr == KindFloat.sentinel(); ok {
		x = math.Float64frombits(v.data)
	}
	return x, ok
}

func Bool(x bool) Value {
	var data uint64
	if x {
		data = 1
	}
	return Value{
		ptr:  KindBool.sentinel(),
		data: data,
	}
}

func (v Value) Bool() (x bool, ok bool) {
	if ok = v.ptr == KindBool.sentinel(); ok {
		x = v.data != 0
	}
	return x, ok
}

func Timestamp(t time.Time) Value {
	return Value{
		ptr:  KindTimestamp.sentinel(),
		data: uint64(t.UnixNano()),
	}
}

func (v Value) Timestamp() (t time.Time, ok bool) {
	if ok = v.ptr == KindTimestamp.sentinel(); ok {
		t = time.Unix(0, int64(v.data))
	}
	return t, ok
}

func SpanId(x [8]byte) Value {
	return Value{
		ptr:  KindSpanId.sentinel(),
		data: binary.LittleEndian.Uint64(x[:]),
	}
}

func (v Value) SpanId() (x [8]byte, ok bool) {
	if ok = v.ptr == KindSpanId.sentinel(); ok {
		binary.LittleEndian.PutUint64(x[:], v.data)
	}
	return x, ok
}

func (v Value) AsAny() (x any) {
	switch v.ptr {
	case KindInt.sentinel():
		x, _ = v.Int()
	case KindUint.sentinel():
		x, _ = v.Uint()
	case KindDuration.sentinel():
		x, _ = v.Duration()
	case KindFloat.sentinel():
		x, _ = v.Float()
	case KindBool.sentinel():
		x, _ = v.Bool()
	case KindTimestamp.sentinel():
		x, _ = v.Timestamp()
	case KindSpanId.sentinel():
		x, _ = v.SpanId()
	default:
		switch v.data >> pointerHiShift {
		case uint64(KindString):
			x, _ = v.String()
		case uint64(KindBytes):
			x, _ = v.Bytes()
		case uint64(KindHistogram):
			x, _ = v.Histogram()
		case uint64(KindTraceId):
			x, _ = v.TraceId()
		}
	}
	return x
}

func (v Value) AppendTo(buf []byte) []byte {
	k := v.Kind()

	buf = rw.AppendUint8(buf, uint8(k))

	switch k {
	case KindEmpty:
		return buf

	case KindString:
		x, _ := v.String()
		buf = rw.AppendVarint(buf, uint64(len(x)))
		buf = rw.AppendString(buf, x)

	case KindBytes:
		x, _ := v.Bytes()
		buf = rw.AppendVarint(buf, uint64(len(x)))
		buf = rw.AppendBytes(buf, x)

	case KindHistogram:
		x, _ := v.Histogram()
		return x.AppendTo(buf)

	case KindTraceId:
		x, _ := v.TraceId()
		buf = rw.AppendBytes(buf, x[:])
		return buf

	case KindSpanId:
		x, _ := v.SpanId()
		buf = rw.AppendBytes(buf, x[:])
		return buf

	default:
		buf = rw.AppendVarint(buf, v.data)
	}

	return buf
}

func (v *Value) ReadFrom(buf []byte) ([]byte, error) {
	*v = Value{} // zero the value for any error paths

	r := rw.NewReader(buf)

	k := Kind(r.ReadUint8())
	if k < KindEmpty || k >= kindLargest {
		return nil, errs.Errorf("invalid kind: %d", k)
	}

	switch k {
	case KindEmpty:

	case KindString:
		*v = String(string(r.ReadBytes(r.ReadVarint())))

	case KindBytes:
		*v = Bytes(bytes.Clone(r.ReadBytes(r.ReadVarint())))

	case KindHistogram:
		rem, err := r.Done()
		if err != nil {
			return nil, err
		}

		h := flathist.NewHistogram()
		rem, err = h.ReadFrom(rem)
		if err != nil {
			return nil, err
		}

		*v = Histogram(h)
		return rem, nil

	case KindTraceId:
		*v = TraceId([16]byte(r.ReadBytes(16)))

	case KindSpanId:
		*v = SpanId([8]byte(r.ReadBytes(8)))

	default:
		*v = Value{
			ptr:  k.sentinel(),
			data: r.ReadVarint(),
		}
	}

	return r.Done()
}
