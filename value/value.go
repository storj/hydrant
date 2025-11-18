package value

import (
	"math"
	"time"
	"unsafe"
)

var sentinels [6]byte

func isSentinel(ptr *byte) bool {
	return uintptr(unsafe.Pointer(ptr)) >= uintptr(unsafe.Pointer(&sentinels[0])) &&
		uintptr(unsafe.Pointer(ptr)) <= uintptr(unsafe.Pointer(&sentinels[5]))
}

type Value struct {
	ptr  *byte
	data uint64
}

func String(x string) (v Value) {
	if uint64(len(x))>>62 == 0 {
		v = Value{
			ptr:  unsafe.StringData(x),
			data: 0b01<<62 | uint64(len(x)),
		}
	}
	return v
}

func (v Value) String() (x string, ok bool) {
	if ok = v.data>>62 == 0b01 && !isSentinel(v.ptr); ok {
		x = unsafe.String(v.ptr, int(v.data&^(0b11<<62)))
	}
	return x, ok
}

func Bytes(x []byte) (v Value) {
	if uint64(len(x))>>62 == 0 {
		v = Value{
			ptr:  unsafe.SliceData(x),
			data: 0b10<<62 | uint64(len(x)),
		}
	}
	return v
}

func (v Value) Bytes() (x []byte, ok bool) {
	if ok = v.data>>62 == 0b10 && !isSentinel(v.ptr); ok {
		x = unsafe.Slice(v.ptr, int(v.data&^(0b11<<62)))
	}
	return x, ok
}

func Int(x int64) Value {
	return Value{
		ptr:  &sentinels[0],
		data: uint64(x),
	}
}

func (v Value) Int() (x int64, ok bool) {
	if ok = v.ptr == &sentinels[0]; ok {
		x = int64(v.data)
	}
	return x, ok
}

func Uint(x uint64) Value {
	return Value{
		ptr:  &sentinels[1],
		data: x,
	}
}

func (v Value) Uint() (x uint64, ok bool) {
	if ok = v.ptr == &sentinels[1]; ok {
		x = v.data
	}
	return x, ok
}

func Duration(x time.Duration) Value {
	return Value{
		ptr:  &sentinels[2],
		data: uint64(x),
	}
}

func (v Value) Duration() (x time.Duration, ok bool) {
	if ok = v.ptr == &sentinels[2]; ok {
		x = time.Duration(v.data)
	}
	return x, ok
}

func Float(x float64) Value {
	return Value{
		ptr:  &sentinels[3],
		data: math.Float64bits(x),
	}
}

func (v Value) Float() (x float64, ok bool) {
	if ok = v.ptr == &sentinels[3]; ok {
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
		ptr:  &sentinels[4],
		data: data,
	}
}

func (v Value) Bool() (x bool, ok bool) {
	if ok = v.ptr == &sentinels[4]; ok {
		x = v.data != 0
	}
	return x, ok
}

func Timestamp(t time.Time) Value {
	return Value{
		ptr:  &sentinels[5],
		data: uint64(t.UnixNano()),
	}
}

func (v Value) Timestamp() (t time.Time, ok bool) {
	if ok = v.ptr == &sentinels[5]; ok {
		t = time.Unix(0, int64(v.data))
	}
	return t, ok
}

func (v Value) AsAny() (x any) {
	switch uintptr(unsafe.Pointer(v.ptr)) - uintptr(unsafe.Pointer(&sentinels[0])) {
	case 0:
		x, _ = v.Int()
	case 1:
		x, _ = v.Uint()
	case 2:
		x, _ = v.Duration()
	case 3:
		x, _ = v.Float()
	case 4:
		x, _ = v.Bool()
	case 5:
		x, _ = v.Timestamp()
	default:
		switch v.data >> 62 {
		case 0b01:
			x, _ = v.String()
		case 0b10:
			x, _ = v.Bytes()
		}
	}
	return x
}
