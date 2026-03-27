// Package processor handles batch processing for enrichment
package processor

import (
	"bytes"
	"fmt"
	"io"
	"sync"
)

const (
	// DefaultFlushThreshold is the default buffer size before auto-flush
	DefaultFlushThreshold = 100
)

// StreamWriter ensures thread-safe streaming with batch prefix and buffering
type StreamWriter struct {
	mu            sync.Mutex
	writer        io.Writer
	buffer        bytes.Buffer
	currentPrefix string
	flushThreshold int
}

// StreamWriterOption configures StreamWriter
type StreamWriterOption func(*StreamWriter)

// WithFlushThreshold sets the buffer flush threshold
func WithFlushThreshold(threshold int) StreamWriterOption {
	return func(sw *StreamWriter) {
		sw.flushThreshold = threshold
	}
}

// NewStreamWriter creates a new StreamWriter.
// It panics if w is nil, as a nil writer would cause runtime errors.
func NewStreamWriter(w io.Writer, opts ...StreamWriterOption) *StreamWriter {
	if w == nil {
		panic("stream writer: writer cannot be nil")
	}
	sw := &StreamWriter{
		writer:         w,
		flushThreshold: DefaultFlushThreshold,
	}
	for _, opt := range opts {
		opt(sw)
	}
	return sw
}

// WriteWithPrefix writes a chunk with the given prefix in a thread-safe manner.
// Chunks are buffered and flushed when:
// - Buffer size exceeds flush threshold
// - Newline is detected in the chunk
// - Flush() is called explicitly
func (sw *StreamWriter) WriteWithPrefix(prefix string, chunk []byte) error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	// If prefix changed, flush existing buffer first
	if sw.currentPrefix != "" && sw.currentPrefix != prefix && sw.buffer.Len() > 0 {
		if err := sw.flushLocked(); err != nil {
			return err
		}
	}
	sw.currentPrefix = prefix

	// Add chunk to buffer
	if _, err := sw.buffer.Write(chunk); err != nil {
		return err
	}

	// Check if we should flush
	if sw.buffer.Len() >= sw.flushThreshold || bytes.Contains(chunk, []byte("\n")) {
		return sw.flushLocked()
	}

	return nil
}

// Flush writes any buffered data to the underlying writer
func (sw *StreamWriter) Flush() error {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.flushLocked()
}

// flushLocked writes buffered data without acquiring lock (caller must hold lock)
func (sw *StreamWriter) flushLocked() error {
	if sw.buffer.Len() == 0 {
		return nil
	}

	// Write prefix and buffered content
	if _, err := fmt.Fprintf(sw.writer, "[%s] ", sw.currentPrefix); err != nil {
		return err
	}
	if _, err := sw.writer.Write(sw.buffer.Bytes()); err != nil {
		return err
	}

	// Reset buffer
	sw.buffer.Reset()
	return nil
}
