package pool

import (
	"math/rand"
	"testing"
	"time"
)

var (
	pool = NewBufferPool(1024)
)

func TestPoolGet(t *testing.T) {
	for i := 0; i <= 1081345; i++ {
		buffer := pool.Get(i)
		b := buffer.Bytes()
		if len(b) != i {
			t.Error("wrong length")
		}
		ri := round(i)
		if cap(b) != ri {
			t.Error("wrong cap", i, cap(b), ri)
		}
		buffer.Release()
	}
}

func BenchmarkBufferPool(b *testing.B) {
	rand.Seed(time.Now().UnixNano())
	maxInt := 1024 * 1024
	for i := 0; i < b.N; i++ {
		buffer := pool.Get(rand.Intn(maxInt))
		buffer.Release()
	}
}

func BenchmarkBufferPoolParallel(b *testing.B) {
	rand.Seed(time.Now().UnixNano())
	maxInt := 1024 * 1024
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buffer := pool.Get(rand.Intn(maxInt))
			buffer.Release()
		}
	})
}

func round(size int) int {
	var offset, spacing int
	if size <= maxSmallBufferSize {
		spacing = smallSpacing
	} else if size <= maxMediumBufferSize {
		offset = maxSmallBufferSize
		spacing = mediumSpacing
	} else {
		offset = maxMediumBufferSize
		spacing = largeSpacing
	}

	n, r := (size-offset)/spacing, (size-offset)%spacing
	if r == 0 {
		n--
	}
	if n > 255 {
		return size
	}
	return offset + (n+1)*spacing
}
