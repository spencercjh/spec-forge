package processor

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamWriter_WriteWithPrefix(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)

	err := sw.WriteWithPrefix("api", []byte("hello"))
	require.NoError(t, err)

	// Buffer hasn't flushed yet (below threshold, no newline)
	assert.Empty(t, buf.String(), "Buffer should not flush immediately for small chunks without newline")

	// Explicit flush should write the content
	err = sw.Flush()
	require.NoError(t, err)

	assert.Equal(t, "[api] hello", buf.String())
}

func TestStreamWriter_WriteWithPrefix_AutoFlushOnNewline(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)

	err := sw.WriteWithPrefix("api", []byte("hello\n"))
	require.NoError(t, err)

	// Should auto-flush on newline
	assert.Equal(t, "[api] hello\n", buf.String())
}

func TestStreamWriter_WriteWithPrefix_AutoFlushOnThreshold(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf, WithFlushThreshold(5))

	// Write exactly at threshold
	err := sw.WriteWithPrefix("api", []byte("12345"))
	require.NoError(t, err)

	// Should flush when reaching threshold
	assert.Contains(t, buf.String(), "[api] 12345", "Buffer should auto-flush when reaching threshold")
}

func TestStreamWriter_ConcurrentWrites(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)

	var wg sync.WaitGroup
	// Use lowercase prefixes to match actual TemplateType values
	prefixes := []string{"api", "schema", "param"}

	for _, prefix := range prefixes {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			_ = sw.WriteWithPrefix(p, []byte("data\n")) // Add newline to trigger flush
		}(prefix)
	}

	wg.Wait()

	output := buf.String()
	// Each prefix should appear exactly once
	assert.Contains(t, output, "[api]")
	assert.Contains(t, output, "[schema]")
	assert.Contains(t, output, "[param]")
	assert.Contains(t, output, "data")
}

func TestStreamWriter_MultipleWritesSamePrefix(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)

	_ = sw.WriteWithPrefix("api", []byte("chunk1"))
	_ = sw.WriteWithPrefix("api", []byte("chunk2"))

	// Explicit flush needed since no newlines
	_ = sw.Flush()

	output := buf.String()
	// Both chunks should be in output with single prefix
	assert.Contains(t, output, "[api] chunk1chunk2")
}

func TestStreamWriter_NilWriterPanics(t *testing.T) {
	assert.Panics(t, func() {
		NewStreamWriter(nil)
	}, "NewStreamWriter should panic when given nil writer")
}

func TestStreamWriter_PrefixChangeFlushesBuffer(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)

	_ = sw.WriteWithPrefix("api", []byte("data1"))
	// Change prefix - should flush previous buffer
	_ = sw.WriteWithPrefix("schema", []byte("data2"))

	// Flush any remaining
	_ = sw.Flush()

	output := buf.String()
	assert.Contains(t, output, "[api] data1")
	assert.Contains(t, output, "[schema] data2")
}

func TestStreamWriter_BatchingReducesOutputCalls(t *testing.T) {
	counter := &writeCounter{calls: 0}
	sw := NewStreamWriter(counter, WithFlushThreshold(100))

	// Write multiple small chunks
	for i := 0; i < 5; i++ {
		err := sw.WriteWithPrefix("api", []byte("x"))
		require.NoError(t, err)
	}

	// No writes yet (buffered, below threshold, no newlines)
	assert.Equal(t, 0, counter.calls, "Should not have written yet due to buffering")

	// Flush explicitly
	err := sw.Flush()
	require.NoError(t, err)

	// Now should have written once (prefix + all buffered content)
	assert.GreaterOrEqual(t, counter.calls, 1, "Should have written after flush")
}

func TestStreamWriter_WithFlushThreshold(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf, WithFlushThreshold(5))

	// Write exactly at threshold
	err := sw.WriteWithPrefix("api", []byte("12345"))
	require.NoError(t, err)

	// Should flush when reaching threshold
	assert.Contains(t, buf.String(), "[api] 12345")
}

func TestStreamWriter_LargeContent(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf, WithFlushThreshold(10))

	// Write content larger than threshold
	largeContent := strings.Repeat("x", 100)
	err := sw.WriteWithPrefix("api", []byte(largeContent))
	require.NoError(t, err)

	// Should have auto-flushed
	assert.Contains(t, buf.String(), "[api]")
	assert.Contains(t, buf.String(), largeContent)
}

func TestStreamWriter_Metrics(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf, WithFlushThreshold(5))

	// Write multiple chunks with different prefixes
	_ = sw.WriteWithPrefix("api", []byte("12345"))    // Triggers flush (threshold)
	_ = sw.WriteWithPrefix("schema", []byte("ab"))   // Buffered (below threshold)
	_ = sw.WriteWithPrefix("schema", []byte("cde"))  // Triggers flush (threshold)
	_ = sw.WriteWithPrefix("param", []byte("xyz\n")) // Triggers flush (newline)

	// Check metrics
	metrics := sw.GetMetrics()
	assert.Equal(t, int64(4), metrics.TotalChunks, "Should have 4 total chunks")
	assert.Equal(t, int64(14), metrics.TotalBytes, "Should have 14 total bytes (5+2+3+4)")
	assert.Equal(t, int64(3), metrics.PrefixCount, "Should have 3 unique prefixes")
	assert.GreaterOrEqual(t, metrics.FlushCount, int64(3), "Should have at least 3 flushes")
}

func TestStreamWriter_WithDebug(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf, WithDebug(true))

	// Write with debug enabled - should not error
	err := sw.WriteWithPrefix("api", []byte("test\n"))
	require.NoError(t, err)

	// Verify output still works
	assert.Contains(t, buf.String(), "[api] test")
}

// writeCounter counts Write calls
type writeCounter struct {
	buf   bytes.Buffer
	calls int
}

func (w *writeCounter) Write(p []byte) (int, error) {
	w.calls++
	return w.buf.Write(p)
}

func (w *writeCounter) String() string {
	return w.buf.String()
}
