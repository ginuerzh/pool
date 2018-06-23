package pool

const (
	MaxSmallBufferSize  = 1024
	MaxMediumBufferSize = 32768
)

type bufferFreeList chan []byte

// Buffer is a byte buffer.
type Buffer struct {
	pool *BufferPool
	data []byte
}

// Bytes returns the byte slice this buffer holds.
func (b *Buffer) Bytes() []byte {
	if b == nil {
		return nil
	}
	return b.data
}

// Release releases the buffer to buffer pool.
func (b *Buffer) Release() {
	if b == nil || b.pool == nil {
		return
	}
	b.pool.Put(b)
}

// BufferPool is a buffer pool.
type BufferPool struct {
	// small buffers are those buffer size less than or equal to 1024 bytes.
	// The buckets are separated by 8 bytes.
	smallBuffers [128]bufferFreeList

	// medium buffers are those buffer size less than or equal to 32K bytes.
	// The buckets are separated by 256 bytes.
	mediumBuffers [124]bufferFreeList

	// large buffers are those buffer size greater than 32K bytes.
	// The buckets are separated by 4K bytes.
	largeBuffers [256]bufferFreeList
}

// NewBufferPool creates a new buffer pool.
// maxSize is the max number of buffers to keep in each pool class.
func NewBufferPool(maxSize int) *BufferPool {
	pool := &BufferPool{}
	for i := range pool.smallBuffers {
		pool.smallBuffers[i] = make(chan []byte, maxSize)
	}
	for i := range pool.mediumBuffers {
		pool.mediumBuffers[i] = make(chan []byte, maxSize)
	}
	for i := range pool.largeBuffers {
		pool.largeBuffers[i] = make(chan []byte, maxSize)
	}
	return pool
}

// Get return a buffer that match the specified size.
func (p *BufferPool) Get(size int) *Buffer {
	if size <= 0 {
		return nil
	}
	if size <= MaxSmallBufferSize {
		return p.getSmallBuffer(size)
	}
	if size <= MaxMediumBufferSize {
		return p.getMediumBuffer(size)
	}
	return p.getLargeBuffer(size)
}

// Put puts the buffer to the buffer pool.
func (p *BufferPool) Put(b *Buffer) {
	if p == nil || b == nil || len(b.data) == 0 {
		return
	}

	size := cap(b.data)
	if size <= MaxSmallBufferSize {
		select {
		case p.smallBuffers[size/8-1] <- b.data:
		default:
		}
		return
	}
	if size <= MaxMediumBufferSize {
		select {
		case p.mediumBuffers[(size-MaxSmallBufferSize)/256-1] <- b.data:
		default:
		}
		return
	}
	select {
	case p.largeBuffers[(size-MaxMediumBufferSize)/4096-1] <- b.data:
	default:
	}
}

func (p *BufferPool) getSmallBuffer(size int) *Buffer {
	n, r := size/8, size%8
	if r == 0 {
		n--
	}

	var buf []byte
	select {
	case buf = <-p.smallBuffers[n]:
	default:
		buf = make([]byte, (n+1)*8)
	}
	return &Buffer{
		data: buf[:size],
		pool: p,
	}
}

func (p *BufferPool) getMediumBuffer(size int) *Buffer {
	n, r := (size-MaxSmallBufferSize)/256, (size-MaxSmallBufferSize)%256
	if r == 0 {
		n--
	}

	var buf []byte
	select {
	case buf = <-p.mediumBuffers[n]:
	default:
		buf = make([]byte, MaxSmallBufferSize+(n+1)*256)
	}
	return &Buffer{
		data: buf[:size],
		pool: p,
	}
}

func (p *BufferPool) getLargeBuffer(size int) *Buffer {
	n, r := (size-MaxMediumBufferSize)/4096, (size-MaxMediumBufferSize)%4096
	if r == 0 {
		n--
	}

	var buf []byte
	select {
	case buf = <-p.largeBuffers[n]:
	default:
		buf = make([]byte, MaxMediumBufferSize+(n+1)*4096)
	}
	return &Buffer{
		data: buf[:size],
		pool: p,
	}
}
