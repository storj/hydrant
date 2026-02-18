package value

func Less(left, right Value) (b bool, ok bool) {
	switch uint64(left.Kind())<<8 | uint64(right.Kind()) {
	default:
		return false, false

	case uint64(KindEmpty)<<8 | uint64(KindEmpty):
		return false, true

	case uint64(KindString)<<8 | uint64(KindString):
		l, _ := left.String()
		r, _ := right.String()
		return l < r, true

	case uint64(KindBytes)<<8 | uint64(KindBytes):
		l, _ := left.Bytes()
		r, _ := right.Bytes()
		return string(l) < string(r), true

	case uint64(KindHistogram)<<8 | uint64(KindHistogram):
		ld, _ := left.Histogram()
		rd, _ := right.Histogram()
		return ld.Min() < rd.Min(), true

	case uint64(KindTraceId)<<8 | uint64(KindTraceId):
		l, _ := left.TraceId()
		r, _ := right.TraceId()
		return string(l[:]) < string(r[:]), true

	case uint64(KindSpanId)<<8 | uint64(KindSpanId):
		l, _ := left.SpanId()
		r, _ := right.SpanId()
		return string(l[:]) < string(r[:]), true

	case uint64(KindInt)<<8 | uint64(KindInt):
		l, _ := left.Int()
		r, _ := right.Int()
		return l < r, true

	case uint64(KindUint)<<8 | uint64(KindUint):
		l, _ := left.Uint()
		r, _ := right.Uint()
		return l < r, true

	case uint64(KindDuration)<<8 | uint64(KindDuration):
		l, _ := left.Duration()
		r, _ := right.Duration()
		return l < r, true

	case uint64(KindFloat)<<8 | uint64(KindFloat):
		l, _ := left.Float()
		r, _ := right.Float()
		return l < r, true

	case uint64(KindBool)<<8 | uint64(KindBool):
		l, _ := left.Bool()
		r, _ := right.Bool()
		return !l && r, true

	case uint64(KindTimestamp)<<8 | uint64(KindTimestamp):
		l, _ := left.Timestamp()
		r, _ := right.Timestamp()
		return l.Before(r), true
	}
}

func Equal(left, right Value) bool {
	if left.ptr == right.ptr {
		return left.data == right.data
	}
	switch (left.data>>pointerHiShift)<<pointerLoShift | (right.data >> pointerHiShift) {
	case 0b001_001:
		lstr, _ := left.String()
		rstr, _ := right.String()
		return lstr == rstr
	case 0b010_010:
		lb, _ := left.Bytes()
		rb, _ := right.Bytes()
		return string(lb) == string(rb)
	case 0b011_011:
		ld, _ := left.Histogram()
		rd, _ := right.Histogram()
		return ld.Equal(rd)
	case 0b100_100:
		lt, _ := left.TraceId()
		rt, _ := right.TraceId()
		return lt == rt
	}
	return false
}
