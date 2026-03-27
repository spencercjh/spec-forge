// Package processor handles batch processing for enrichment
package processor

import (
	"fmt"
	"io"
	"sync"
)

// StreamWriter ensures thread-safe streaming with batch prefix
type StreamWriter struct {
	mu     sync.Mutex
	writer io.Writer
}

// NewStreamWriter creates a new StreamWriter
func NewStreamWriter(w io.Writer) *StreamWriter {
	return &StreamWriter{writer: w}
}

// WriteWithPrefix writes a chunk with the given prefix in a thread-safe manner
func (sw *StreamWriter) WriteWithPrefix(prefix string, chunk []byte) error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	_, err := fmt.Fprintf(sw.writer, "[%s] ", prefix)
	if err != nil {
		return err
	}
	_, err = sw.writer.Write(chunk)
	return err
}
