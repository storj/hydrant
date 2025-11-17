package value

import (
	"math"
	"time"
	"unsafe"
)

var sentinels [8]byte

type Value struct {
	ptr1 unsafe.Pointer
	ptr2 unsafe.Pointer
	data uint64
}

func String(x string) Value {
	return Value{
		ptr1: unsafe.Pointer(&sentinels[0]),
		ptr2: unsafe.Pointer(unsafe.StringData(x)),
		data: uint64(len(x)),
	}
}

func (v Value) String() (x string, ok bool) {
	ok = v.ptr1 == unsafe.Pointer(&sentinels[0])
	if ok {
		x = unsafe.String((*byte)(v.ptr2), int(v.data))
	}
	return x, ok
}

func Bytes(x []byte) Value {
	return Value{
		ptr1: unsafe.Pointer(&sentinels[1]),
		ptr2: unsafe.Pointer(unsafe.SliceData(x)),
		data: uint64(len(x)),
	}
}

func (v Value) Bytes() (x []byte, ok bool) {
	ok = v.ptr1 == unsafe.Pointer(&sentinels[1])
	if ok {
		x = unsafe.Slice((*byte)(v.ptr2), int(v.data))
	}
	return x, ok
}

func Int(x int64) Value {
	return Value{
		ptr1: unsafe.Pointer(&sentinels[2]),
		data: uint64(x),
	}
}

func (v Value) Int() (x int64, ok bool) {
	ok = v.ptr1 == unsafe.Pointer(&sentinels[2])
	if ok {
		x = int64(v.data)
	}
	return x, ok
}

func Uint(x uint64) Value {
	return Value{
		ptr1: unsafe.Pointer(&sentinels[3]),
		data: x,
	}
}

func (v Value) Uint() (x uint64, ok bool) {
	ok = v.ptr1 == unsafe.Pointer(&sentinels[3])
	if ok {
		x = v.data
	}
	return x, ok
}

func Duration(x time.Duration) Value {
	return Value{
		ptr1: unsafe.Pointer(&sentinels[4]),
		data: uint64(x),
	}
}

func (v Value) Duration() (x time.Duration, ok bool) {
	ok = v.ptr1 == unsafe.Pointer(&sentinels[4])
	if ok {
		x = time.Duration(v.data)
	}
	return x, ok
}

func Float(x float64) Value {
	return Value{
		ptr1: unsafe.Pointer(&sentinels[5]),
		data: math.Float64bits(x),
	}
}

func (v Value) Float() (x float64, ok bool) {
	ok = v.ptr1 == unsafe.Pointer(&sentinels[5])
	if ok {
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
		ptr1: unsafe.Pointer(&sentinels[6]),
		data: data,
	}
}

func (v Value) Bool() (x bool, ok bool) {
	ok = v.ptr1 == unsafe.Pointer(&sentinels[6])
	if ok {
		x = v.data != 0
	}
	return x, ok
}

func Timestamp(t time.Time) Value {
	return Value{
		ptr1: unsafe.Pointer(&sentinels[7]),
		data: uint64(t.UnixNano()),
	}
}

func (v Value) Timestamp() (t time.Time, ok bool) {
	ok = v.ptr1 == unsafe.Pointer(&sentinels[7])
	if ok {
		t = time.Unix(0, int64(v.data))
	}
	return t, ok
}

func Any(x any) Value {
	i := (*[2]unsafe.Pointer)(unsafe.Pointer(&x))
	return Value{
		ptr1: i[0],
		ptr2: i[1],
	}
}

func (v Value) AsAny() (x any) {
	switch v.ptr1 {
	case unsafe.Pointer(&sentinels[0]):
		x, _ = v.String()
	case unsafe.Pointer(&sentinels[1]):
		x, _ = v.Bytes()
	case unsafe.Pointer(&sentinels[2]):
		x, _ = v.Int()
	case unsafe.Pointer(&sentinels[3]):
		x, _ = v.Uint()
	case unsafe.Pointer(&sentinels[4]):
		x, _ = v.Duration()
	case unsafe.Pointer(&sentinels[5]):
		x, _ = v.Float()
	case unsafe.Pointer(&sentinels[6]):
		x, _ = v.Bool()
	case unsafe.Pointer(&sentinels[7]):
		x, _ = v.Timestamp()
	default:
		*(*[2]unsafe.Pointer)(unsafe.Pointer(&x)) = [2]unsafe.Pointer{v.ptr1, v.ptr2}
	}
	return x
}
