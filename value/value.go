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

func OfString(x string) Value {
	return Value{
		ptr1: unsafe.Pointer(&sentinels[0]),
		ptr2: unsafe.Pointer(unsafe.StringData(x)),
		data: uint64(len(x)),
	}
}

func (v Value) AsString() (x string, ok bool) {
	ok = v.ptr1 == unsafe.Pointer(&sentinels[0])
	if ok {
		x = unsafe.String((*byte)(v.ptr2), int(v.data))
	}
	return x, ok
}

func OfBytes(x []byte) Value {
	return Value{
		ptr1: unsafe.Pointer(&sentinels[1]),
		ptr2: unsafe.Pointer(unsafe.SliceData(x)),
		data: uint64(len(x)),
	}
}

func (v Value) AsBytes() (x []byte, ok bool) {
	ok = v.ptr1 == unsafe.Pointer(&sentinels[1])
	if ok {
		x = unsafe.Slice((*byte)(v.ptr2), int(v.data))
	}
	return x, ok
}

func OfInt(x int64) Value {
	return Value{
		ptr1: unsafe.Pointer(&sentinels[2]),
		data: uint64(x),
	}
}

func (v Value) AsInt() (x int64, ok bool) {
	ok = v.ptr1 == unsafe.Pointer(&sentinels[2])
	if ok {
		x = int64(v.data)
	}
	return x, ok
}

func OfUint(x uint64) Value {
	return Value{
		ptr1: unsafe.Pointer(&sentinels[3]),
		data: x,
	}
}

func (v Value) AsUint() (x uint64, ok bool) {
	ok = v.ptr1 == unsafe.Pointer(&sentinels[3])
	if ok {
		x = v.data
	}
	return x, ok
}

func OfDuration(x time.Duration) Value {
	return Value{
		ptr1: unsafe.Pointer(&sentinels[4]),
		data: uint64(x),
	}
}

func (v Value) AsDuration() (x time.Duration, ok bool) {
	ok = v.ptr1 == unsafe.Pointer(&sentinels[4])
	if ok {
		x = time.Duration(v.data)
	}
	return x, ok
}

func OfFloat(x float64) Value {
	return Value{
		ptr1: unsafe.Pointer(&sentinels[5]),
		data: math.Float64bits(x),
	}
}

func (v Value) AsFloat() (x float64, ok bool) {
	ok = v.ptr1 == unsafe.Pointer(&sentinels[5])
	if ok {
		x = math.Float64frombits(v.data)
	}
	return x, ok
}

func OfBool(x bool) Value {
	var data uint64
	if x {
		data = 1
	}
	return Value{
		ptr1: unsafe.Pointer(&sentinels[6]),
		data: data,
	}
}

func (v Value) AsBool() (x bool, ok bool) {
	ok = v.ptr1 == unsafe.Pointer(&sentinels[6])
	if ok {
		x = v.data != 0
	}
	return x, ok
}

func OfTimestamp(t time.Time) Value {
	return Value{
		ptr1: unsafe.Pointer(&sentinels[7]),
		data: uint64(t.UnixNano()),
	}
}

func (v Value) AsTimestamp() (t time.Time, ok bool) {
	ok = v.ptr1 == unsafe.Pointer(&sentinels[7])
	if ok {
		t = time.Unix(0, int64(v.data))
	}
	return t, ok
}

func OfAny(x any) Value {
	i := (*[2]unsafe.Pointer)(unsafe.Pointer(&x))
	return Value{
		ptr1: i[0],
		ptr2: i[1],
	}
}

func (v Value) AsAny() (x any) {
	switch v.ptr1 {
	case unsafe.Pointer(&sentinels[0]):
		x, _ = v.AsString()
	case unsafe.Pointer(&sentinels[1]):
		x, _ = v.AsBytes()
	case unsafe.Pointer(&sentinels[2]):
		x, _ = v.AsInt()
	case unsafe.Pointer(&sentinels[3]):
		x, _ = v.AsUint()
	case unsafe.Pointer(&sentinels[4]):
		x, _ = v.AsDuration()
	case unsafe.Pointer(&sentinels[5]):
		x, _ = v.AsFloat()
	case unsafe.Pointer(&sentinels[6]):
		x, _ = v.AsBool()
	case unsafe.Pointer(&sentinels[7]):
		x, _ = v.AsTimestamp()
	default:
		*(*[2]unsafe.Pointer)(unsafe.Pointer(&x)) = [2]unsafe.Pointer{v.ptr1, v.ptr2}
	}
	return x
}
