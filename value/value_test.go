package value

import (
	"fmt"
	"math"
	"testing"
	"time"
	"unsafe"

	"github.com/zeebo/assert"
)

var g byte

func TestIsSentinel(t *testing.T) {
	for i := range len(sentinels) {
		assert.That(t, isSentinel(&sentinels[i]))
	}

	assert.That(t, !isSentinel(nil))
	assert.That(t, !isSentinel(&g))

	var b byte
	assert.That(t, !isSentinel(&b))

	assert.That(t, !isSentinel((*byte)(unsafe.Add(unsafe.Pointer(&sentinels[len(sentinels)-1]), 1))))
	assert.That(t, !isSentinel((*byte)(unsafe.Add(unsafe.Pointer(&sentinels[0]), -1))))
}

func TestKind(t *testing.T) {
	assert.Equal(t, String("hello").Kind(), KindString)
	assert.Equal(t, Bytes([]byte{1, 2, 3}).Kind(), KindBytes)
	assert.Equal(t, Int(42).Kind(), KindInt)
	assert.Equal(t, Uint(42).Kind(), KindUint)
	assert.Equal(t, Duration(5*time.Second).Kind(), KindDuration)
	assert.Equal(t, Float(3.14).Kind(), KindFloat)
	assert.Equal(t, Bool(true).Kind(), KindBool)
	assert.Equal(t, Timestamp(time.Now()).Kind(), KindTimestamp)
	assert.Equal(t, Identifier(65).Kind(), KindIdentifier)
}

func TestValue(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		runValueTest(t, String, Value.String,
			"hello world",
			"",
			"Hello 世界 🌍",
		)
	})

	t.Run("bytes", func(t *testing.T) {
		runValueTest(t, Bytes, Value.Bytes,
			[]byte{1, 2, 3, 4, 5},
			[]byte{},
			nil,
			[]byte("Hello 世界 🌍"),
		)
	})

	t.Run("int", func(t *testing.T) {
		runValueTest(t, Int, Value.Int,
			int64(42),
			int64(-12345),
			int64(0),
			int64(math.MaxInt64),
			int64(math.MinInt64),
		)
	})

	t.Run("uint", func(t *testing.T) {
		runValueTest(t, Uint, Value.Uint,
			uint64(42),
			uint64(0),
			uint64(math.MaxUint64),
		)
	})

	t.Run("duration", func(t *testing.T) {
		runValueTest(t, Duration, Value.Duration,
			5*time.Second,
			-10*time.Minute,
			0,
			123*time.Nanosecond,
			456*time.Microsecond,
			789*time.Millisecond,
			24*time.Hour,
			time.Duration(math.MaxInt64),
			time.Duration(math.MinInt64),
			2*time.Hour+30*time.Minute+45*time.Second,
		)
	})

	t.Run("float", func(t *testing.T) {
		runValueTest(t, Float, Value.Float,
			3.14159,
			-2.71828,
			0.0,
			math.Copysign(0, -1),
			math.Inf(1),
			math.Inf(-1),
			math.MaxFloat64,
			math.SmallestNonzeroFloat64,
		)
	})

	t.Run("bool", func(t *testing.T) {
		runValueTest(t, Bool, Value.Bool,
			true,
			false,
		)
	})

	t.Run("timestamp", func(t *testing.T) {
		runValueTest(t, Timestamp, Value.Timestamp,
			time.Now(),
			time.Now().UTC(),
			time.Unix(0, 0),
			time.Now().Add(100*time.Hour),
			time.Now().Add(-100*time.Hour),
		)
	})

	t.Run("identifier", func(t *testing.T) {
		runValueTest(t, Identifier, Value.Identifier,
			uint64(42),
			uint64(0),
			uint64(math.MaxUint64),
		)
	})
}

func TestValueAsAnyOptimizedTypes(t *testing.T) {
	nonNilEmptyString := "a"[1:]
	t.Log("non-nil empty string pointer is non-nil:", unsafe.StringData(nonNilEmptyString) != nil)

	t.Run("string", func(t *testing.T) { assertAsAnyType(t, String("hello"), "hello") })
	t.Run("empty string", func(t *testing.T) { assertAsAnyType(t, String(""), "") })
	t.Run("non-nil empty string", func(t *testing.T) { assertAsAnyType(t, String(nonNilEmptyString), "") })
	t.Run("bytes", func(t *testing.T) { assertAsAnyType(t, Bytes([]byte{1, 2, 3}), []byte{1, 2, 3}) })
	t.Run("empty bytes", func(t *testing.T) { assertAsAnyType(t, Bytes([]byte{}), []byte{}) })
	t.Run("nil bytes", func(t *testing.T) { assertAsAnyType(t, Bytes(nil), []byte(nil)) })
	t.Run("int", func(t *testing.T) { assertAsAnyType(t, Int(42), int64(42)) })
	t.Run("uint", func(t *testing.T) { assertAsAnyType(t, Uint(42), uint64(42)) })
	t.Run("duration", func(t *testing.T) { assertAsAnyType(t, Duration(5*time.Second), 5*time.Second) })
	t.Run("float", func(t *testing.T) { assertAsAnyType(t, Float(3.14), 3.14) })
	t.Run("bool", func(t *testing.T) { assertAsAnyType(t, Bool(true), true) })
	t.Run("timestamp", func(t *testing.T) { assertAsAnyType(t, Timestamp(time.Unix(1, 2)), time.Unix(1, 2)) })
	t.Run("identifier", func(t *testing.T) { assertAsAnyType(t, Identifier(65), uint64(65)) })
}

func TestValueZeroValue(t *testing.T) {
	var v Value

	// Zero value should not match any constructed type
	assertNotType(t, v, Value.String)
	assertNotType(t, v, Value.Bytes)
	assertNotType(t, v, Value.Int)
	assertNotType(t, v, Value.Uint)
	assertNotType(t, v, Value.Duration)
	assertNotType(t, v, Value.Float)
	assertNotType(t, v, Value.Bool)
	assertNotType(t, v, Value.Timestamp)
	assertNotType(t, v, Value.Identifier)

	// AsAny should return nil interface
	assert.Nil(t, v.AsAny())
}

func TestSerialize(t *testing.T) {
	t.Logf("%x", Value{}.Serialize(nil))
	t.Logf("%x", String("hello").Serialize(nil))
	t.Logf("%x", Bytes([]byte{1, 2, 3, 4, 5}).Serialize(nil))
	t.Logf("%x", Int(42).Serialize(nil))
	t.Logf("%x", Uint(42).Serialize(nil))
	t.Logf("%x", Duration(5*time.Second).Serialize(nil))
	t.Logf("%x", Float(3.14).Serialize(nil))
	t.Logf("%x", Bool(true).Serialize(nil))
	t.Logf("%x", Timestamp(time.Unix(1, 2)).Serialize(nil))
	t.Logf("%x", Identifier(64).Serialize(nil))
}

//
// benchmarks
//

func BenchmarkValue(b *testing.B) {
	b.Run("string", func(b *testing.B) { benchmarkType(b, "hello world", String, Value.String) })
	b.Run("bytes", func(b *testing.B) { benchmarkType(b, []byte("hello world"), Bytes, Value.Bytes) })
	b.Run("int", func(b *testing.B) { benchmarkType(b, int64(42), Int, Value.Int) })
	b.Run("uint", func(b *testing.B) { benchmarkType(b, uint64(42), Uint, Value.Uint) })
	b.Run("float", func(b *testing.B) { benchmarkType(b, 3.14159, Float, Value.Float) })
	b.Run("duration", func(b *testing.B) { benchmarkType(b, 5*time.Second, Duration, Value.Duration) })
	b.Run("bool", func(b *testing.B) { benchmarkType(b, true, Bool, Value.Bool) })
	b.Run("timestamp", func(b *testing.B) { benchmarkType(b, time.Now(), Timestamp, Value.Timestamp) })
	b.Run("identifier", func(b *testing.B) { benchmarkType(b, uint64(65), Identifier, Value.Identifier) })
}

func BenchmarkValueAsAnyFromPrimitive(b *testing.B) {
	run := func(b *testing.B, v Value) {
		b.ReportAllocs()
		for b.Loop() {
			_ = v.AsAny()
		}
	}

	b.Run("zero", func(b *testing.B) { run(b, Value{}) })
	b.Run("string", func(b *testing.B) { run(b, String("hello world")) })
	b.Run("bytes", func(b *testing.B) { run(b, Bytes([]byte("hello world"))) })
	b.Run("int", func(b *testing.B) { run(b, Int(4200)) })
	b.Run("uint", func(b *testing.B) { run(b, Uint(4200)) })
	b.Run("float", func(b *testing.B) { run(b, Float(3.14)) })
	b.Run("duration", func(b *testing.B) { run(b, Duration(5*time.Second)) })
	b.Run("bool", func(b *testing.B) { run(b, Bool(true)) })
	b.Run("timestamp", func(b *testing.B) { run(b, Timestamp(time.Now())) })
	b.Run("identifier", func(b *testing.B) { run(b, Identifier(4200)) })
}

//
// helpers
//

func assertEqual[T any](t *testing.T, expected, actual T) {
	switch exp := any(expected).(type) {
	case time.Time:
		assert.True(t, exp.Equal(any(actual).(time.Time)))
	default:
		assert.Equal(t, actual, expected)
	}
}

func runValueTest[T any](t *testing.T, of func(T) Value, as func(Value) (T, bool), samples ...T) {
	t.Helper()
	for _, sample := range samples {
		t.Run(fmt.Sprint(sample), func(t *testing.T) {
			t.Helper()
			var result T
			var ok bool
			allocs := testing.AllocsPerRun(100, func() { result, ok = as(of(sample)) })
			assert.That(t, ok)
			assertEqual(t, sample, result)
			assert.Equal(t, allocs, 0.0)
		})
	}
}

func assertAsAnyType[T any](t *testing.T, v Value, expected T) {
	result := v.AsAny()
	actual, ok := result.(T)
	assert.That(t, ok)
	assertEqual(t, expected, actual)
}

func assertNotType[T any](t *testing.T, v Value, as func(Value) (T, bool)) {
	t.Helper()
	_, ok := as(v)
	assert.That(t, !ok)
}

func benchmarkType[T any](b *testing.B, sample T, of func(T) Value, as func(Value) (T, bool)) {
	b.Run("of", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = of(sample)
		}
	})

	b.Run("as", func(b *testing.B) {
		b.ReportAllocs()
		v := of(sample)
		for b.Loop() {
			_, _ = as(v)
		}
	})
}

type testStringer struct {
	value string
}

func (ts *testStringer) String() string {
	return ts.value
}
