// Package processor handles batch processing for enrichment
package processor

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
)

const (
	// DefaultFlushThreshold is the default buffer size before auto-flush
	DefaultFlushThreshold = 100
)

// StreamWriterMetrics holds streaming statistics
type StreamWriterMetrics struct {
	TotalChunks int64 // Total number of chunks written
	TotalBytes  int64 // Total bytes written (excluding prefixes)
	FlushCount  int64 // Number of flush operations
	PrefixCount int64 // Number of unique prefixes used
}

// StreamWriter ensures thread-safe streaming with batch prefix and buffering
type StreamWriter struct {
	mu             sync.Mutex
	writer         io.Writer
	buffer         bytes.Buffer
	currentPrefix  string
	flushThreshold int
	debug          bool
	metrics        StreamWriterMetrics
	prefixSet      map[string]bool
}

// StreamWriterOption configures StreamWriter
type StreamWriterOption func(*StreamWriter)

// WithFlushThreshold sets the buffer flush threshold
func WithFlushThreshold(threshold int) StreamWriterOption {
	return func(sw *StreamWriter) {
		sw.flushThreshold = threshold
	}
}

// WithDebug enables debug logging for streaming events
func WithDebug(debug bool) StreamWriterOption {
	return func(sw *StreamWriter) {
		sw.debug = debug
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
		prefixSet:      make(map[string]bool),
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

	// Update metrics
	atomic.AddInt64(&sw.metrics.TotalChunks, 1)
	atomic.AddInt64(&sw.metrics.TotalBytes, int64(len(chunk)))

	// Track unique prefixes
	if !sw.prefixSet[prefix] {
		sw.prefixSet[prefix] = true
		atomic.AddInt64(&sw.metrics.PrefixCount, 1)
	}

	// Debug logging
	if sw.debug {
		slog.Debug("stream chunk received",
			"prefix", prefix,
			"chunk_size", len(chunk),
			"buffer_size", sw.buffer.Len(),
			"total_chunks", atomic.LoadInt64(&sw.metrics.TotalChunks),
			"total_bytes", atomic.LoadInt64(&sw.metrics.TotalBytes),
		)
	}

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
	if sw.buffer.Len() >= sw.flushThreshold || bytes.IndexByte(chunk, '\n') >= 0 {
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

	atomic.AddInt64(&sw.metrics.FlushCount, 1)

	// Debug logging
	if sw.debug {
		slog.Debug("stream buffer flushed",
			"prefix", sw.currentPrefix,
			"buffer_size", sw.buffer.Len(),
			"flush_count", atomic.LoadInt64(&sw.metrics.FlushCount),
		)
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

// GetMetrics returns current streaming metrics
func (sw *StreamWriter) GetMetrics() StreamWriterMetrics {
	return StreamWriterMetrics{
		TotalChunks: atomic.LoadInt64(&sw.metrics.TotalChunks),
		TotalBytes:  atomic.LoadInt64(&sw.metrics.TotalBytes),
		FlushCount:  atomic.LoadInt64(&sw.metrics.FlushCount),
		PrefixCount: atomic.LoadInt64(&sw.metrics.PrefixCount),
	}
}
