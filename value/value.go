package value

import (
	"bytes"
	"encoding/binary"
	"math"
	"time"
	"unsafe"

	"github.com/zeebo/errs/v2"

	"github.com/histdb/histdb/flathist"
)

var sentinels [7]byte

type Kind uint8

const (
	KindEmpty      Kind = 0
	KindString     Kind = 1
	KindBytes      Kind = 2
	KindHistogram  Kind = 3
	KindInt        Kind = 4
	KindUint       Kind = 5
	KindDuration   Kind = 6
	KindFloat      Kind = 7
	KindBool       Kind = 8
	KindTimestamp  Kind = 9
	KindIdentifier Kind = 10
)

func (k Kind) sentinel() unsafe.Pointer {
	return unsafe.Pointer(&sentinels[k-4])
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
		return Kind(d + 4)
	}
	return Kind(v.data >> 62)
}

func String(x string) (v Value) {
	if uint64(len(x))>>62 == 0 {
		v = Value{
			ptr:  unsafe.Pointer(unsafe.StringData(x)),
			data: 0b01<<62 | uint64(len(x)),
		}
	}
	return v
}

func (v Value) String() (x string, ok bool) {
	if ok = v.data>>62 == 0b01 && !isSentinel(v.ptr); ok {
		x = unsafe.String((*byte)(v.ptr), int(v.data&^(0b11<<62)))
	}
	return x, ok
}

func Bytes(x []byte) (v Value) {
	if uint64(len(x))>>62 == 0 {
		v = Value{
			ptr:  unsafe.Pointer(unsafe.SliceData(x)),
			data: 0b10<<62 | uint64(len(x)),
		}
	}
	return v
}

func (v Value) Bytes() (x []byte, ok bool) {
	if ok = v.data>>62 == 0b10 && !isSentinel(v.ptr); ok {
		x = unsafe.Slice((*byte)(v.ptr), int(v.data&^(0b11<<62)))
	}
	return x, ok
}

func Histogram(x *flathist.Histogram) (v Value) {
	if x != nil {
		v = Value{
			ptr:  unsafe.Pointer(x),
			data: 0b11 << 62,
		}
	}
	return v
}

func (v Value) Histogram() (x *flathist.Histogram, ok bool) {
	if ok = v.data>>62 == 0b11 && !isSentinel(v.ptr); ok {
		x = (*flathist.Histogram)(v.ptr)
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

func Identifier(x uint64) Value {
	return Value{
		ptr:  KindIdentifier.sentinel(),
		data: x,
	}
}

func (v Value) Identifier() (x uint64, ok bool) {
	if ok = v.ptr == KindIdentifier.sentinel(); ok {
		x = v.data
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
	case KindIdentifier.sentinel():
		x, _ = v.Identifier()
	default:
		switch v.data >> 62 {
		case 0b01:
			x, _ = v.String()
		case 0b10:
			x, _ = v.Bytes()
		case 0b11:
			x, _ = v.Histogram()
		}
	}
	return x
}

func (v Value) AppendTo(buf []byte) []byte {
	k := v.Kind()

	buf = append(buf, byte(k))
	switch k {
	case KindHistogram:
		x, _ := v.Histogram()
		return x.AppendTo(buf)

	case KindEmpty:
		return buf
	}

	var tmp [8]byte
	binary.LittleEndian.PutUint64(tmp[:], v.data)
	buf = append(buf, tmp[:]...)

	switch k {
	case KindString:
		x, _ := v.String()
		buf = append(buf, x...)

	case KindBytes:
		x, _ := v.Bytes()
		buf = append(buf, x...)
	}

	return buf
}

func (v *Value) ReadFrom(buf []byte) ([]byte, error) {
	*v = Value{} // zero the value for any error paths

	r := reader{buf: buf}

	k := Kind(r.ReadUint8())
	if k < KindEmpty || k > KindIdentifier {
		return nil, errs.Errorf("invalid kind: %d", k)
	}

	switch k {
	case KindEmpty:

	case KindString:
		n := r.ReadUint64() &^ (0b11 << 62)
		*v = String(string(r.ReadBytes(n)))

	case KindBytes:
		n := r.ReadUint64() &^ (0b11 << 62)
		*v = Bytes(bytes.Clone(r.ReadBytes(n)))

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

	default:
		*v = Value{
			ptr:  k.sentinel(),
			data: r.ReadUint64(),
		}
	}

	return r.Done()
}
