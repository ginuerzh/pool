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
	for i := 0; i < 1048576; i++ {
		buffer := pool.Get(i)
		b := buffer.Bytes()
		if len(b) != i {
			t.Error("wrong length")
		}
		buffer.Release()
	}
}

func BenchmarkBufferPool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buffer := pool.Get(i)
		buffer.Release()
	}
}

func BenchmarkBufferPoolParallel(b *testing.B) {
	rand.Seed(time.Now().UnixNano())
	maxInt := 128 * 1024
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buffer := pool.Get(rand.Intn(maxInt))
			buffer.Release()
		}
	})
}
