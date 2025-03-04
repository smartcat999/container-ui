package proxy

import (
    "bufio"
    "io"
)

// BufferedReadCloser 带缓冲的ReadCloser
type BufferedReadCloser struct {
    ReadCloser io.ReadCloser
    bufferSize int
    buffer     *bufio.Reader
}

// NewBufferedReadCloser 创建新的带缓冲的ReadCloser
func NewBufferedReadCloser(rc io.ReadCloser, size int) *BufferedReadCloser {
    return &BufferedReadCloser{
        ReadCloser: rc,
        bufferSize: size,
    }
}

func (b *BufferedReadCloser) Read(p []byte) (int, error) {
    if b.buffer == nil {
        b.buffer = bufio.NewReaderSize(b.ReadCloser, b.bufferSize)
    }
    return b.buffer.Read(p)
}

func (b *BufferedReadCloser) Close() error {
    return b.ReadCloser.Close()
}