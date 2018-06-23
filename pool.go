package pool

const (
	maxSmallBufferSize  = 1024
	maxMediumBufferSize = 32768
)

const (
	smallSpacing  = 8
	mediumSpacing = 256
	largeSpacing  = 4096
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
	b.pool = nil
	b.data = nil
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
	if size <= maxSmallBufferSize {
		return p.getSmallBuffer(size)
	}
	if size <= maxMediumBufferSize {
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
	if size <= maxSmallBufferSize {
		select {
		case p.smallBuffers[size/smallSpacing-1] <- b.data:
		default:
		}
		return
	}
	if size <= maxMediumBufferSize {
		select {
		case p.mediumBuffers[(size-maxSmallBufferSize)/mediumSpacing-1] <- b.data:
		default:
		}
		return
	}
	select {
	case p.largeBuffers[(size-maxMediumBufferSize)/largeSpacing-1] <- b.data:
	default:
	}
}

func (p *BufferPool) getSmallBuffer(size int) *Buffer {
	n, r := size/smallSpacing, size%smallSpacing
	if r == 0 {
		n--
	}

	var buf []byte
	select {
	case buf = <-p.smallBuffers[n]:
	default:
		buf = make([]byte, (n+1)*smallSpacing)
	}
	return &Buffer{
		data: buf[:size],
		pool: p,
	}
}

func (p *BufferPool) getMediumBuffer(size int) *Buffer {
	n, r := (size-maxSmallBufferSize)/mediumSpacing, (size-maxSmallBufferSize)%mediumSpacing
	if r == 0 {
		n--
	}

	var buf []byte
	select {
	case buf = <-p.mediumBuffers[n]:
	default:
		buf = make([]byte, maxSmallBufferSize+(n+1)*mediumSpacing)
	}
	return &Buffer{
		data: buf[:size],
		pool: p,
	}
}

func (p *BufferPool) getLargeBuffer(size int) *Buffer {
	n, r := (size-maxMediumBufferSize)/largeSpacing, (size-maxMediumBufferSize)%largeSpacing
	if r == 0 {
		n--
	}
	if n > 255 {
		return &Buffer{
			data: make([]byte, size),
		}
	}

	var buf []byte
	select {
	case buf = <-p.largeBuffers[n]:
	default:
		buf = make([]byte, maxMediumBufferSize+(n+1)*largeSpacing)
	}
	return &Buffer{
		data: buf[:size],
		pool: p,
	}
}
