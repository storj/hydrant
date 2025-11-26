package floathist

import (
	"fmt"
	"math/bits"
	"sync/atomic"
)

type bit64 struct{ b uint64 }

func new64(v uint64) bit64 { return bit64{v} }

func (b *bit64) AtomicClone() bit64      { return bit64{atomic.LoadUint64(&b.b)} }
func (b *bit64) AtomicAddIdx(idx uint)   { atomic.AddUint64(&b.b, 1<<(idx&63)) }
func (b *bit64) AtomicHas(idx uint) bool { return atomic.LoadUint64(&b.b)&(1<<(idx&63)) > 0 }
func (b *bit64) ClearLowest()            { b.b &= b.b - 1 }
func (b bit64) Empty() bool              { return b.b == 0 }
func (b bit64) Lowest() uint             { return uint(bits.TrailingZeros64(b.b)) % 64 }
func (b bit64) Highest() uint            { return uint(63-bits.LeadingZeros64(b.b)) % 64 }
func (b bit64) String() string           { return fmt.Sprintf("%064b", b.b) }

type bit32 struct{ b uint32 }

func new32(v uint32) bit32 { return bit32{v} }

func (b *bit32) AtomicClone() bit32      { return bit32{atomic.LoadUint32(&b.b)} }
func (b *bit32) AtomicAddIdx(idx uint)   { atomic.AddUint32(&b.b, 1<<(idx&31)) }
func (b *bit32) AtomicHas(idx uint) bool { return atomic.LoadUint32(&b.b)&(1<<(idx&31)) > 0 }
func (b *bit32) ClearLowest()            { b.b &= b.b - 1 }
func (b bit32) Empty() bool              { return b.b == 0 }
func (b bit32) Lowest() uint             { return uint(bits.TrailingZeros32(b.b)) % 32 }
func (b bit32) Highest() uint            { return uint(31-bits.LeadingZeros32(b.b)) % 32 }
func (b bit32) String() string           { return fmt.Sprintf("%032b", b.b) }
