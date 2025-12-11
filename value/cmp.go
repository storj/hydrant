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

	case uint64(KindIdentifier)<<8 | uint64(KindIdentifier):
		l, _ := left.Identifier()
		r, _ := right.Identifier()
		return l < r, true
	}
}

func Equal(left, right Value) bool {
	if left.ptr == right.ptr {
		return left.data == right.data
	}
	switch (left.data>>62)<<2 | (right.data >> 62) {
	case 0b0101:
		lstr, _ := left.String()
		rstr, _ := right.String()
		return lstr == rstr
	case 0b1010:
		lb, _ := left.Bytes()
		rb, _ := right.Bytes()
		return string(lb) == string(rb)
	case 0b1111:
		ld, _ := left.Histogram()
		rd, _ := right.Histogram()
		return ld.Equal(rd)
	}
	return false
}
