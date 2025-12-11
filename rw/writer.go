package rw

import (
	"encoding/binary"

	"github.com/histdb/histdb/varint"
)

func AppendVarint(buf []byte, x uint64) []byte {
	var tmp [9]byte
	nb := varint.Append(&tmp, x)
	return append(buf, tmp[:nb]...)
}

func AppendUint8(buf []byte, x uint8) []byte {
	return append(buf, x)
}

func AppendUint64(buf []byte, x uint64) []byte {
	var tmp [8]byte
	binary.LittleEndian.PutUint64(tmp[:], x)
	return append(buf, tmp[:]...)
}

func AppendBytes(buf []byte, x []byte) []byte {
	return append(buf, x...)
}

func AppendString(buf []byte, x string) []byte {
	return append(buf, x...)
}
