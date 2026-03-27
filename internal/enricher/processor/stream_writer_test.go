package processor

import (
	"bytes"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamWriter_WriteWithPrefix(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)

	err := sw.WriteWithPrefix("API", []byte("hello"))
	require.NoError(t, err)

	assert.Equal(t, "[API] hello", buf.String())
}

func TestStreamWriter_ConcurrentWrites(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)

	var wg sync.WaitGroup
	prefixes := []string{"API", "Schema", "Param"}

	for _, prefix := range prefixes {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			_ = sw.WriteWithPrefix(p, []byte("data"))
		}(prefix)
	}

	wg.Wait()

	output := buf.String()
	// Each prefix should appear exactly once
	assert.Contains(t, output, "[API]")
	assert.Contains(t, output, "[Schema]")
	assert.Contains(t, output, "[Param]")
	assert.Contains(t, output, "data")
}

func TestStreamWriter_MultipleWritesSamePrefix(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)

	_ = sw.WriteWithPrefix("API", []byte("chunk1"))
	_ = sw.WriteWithPrefix("API", []byte("chunk2"))

	output := buf.String()
	assert.Contains(t, output, "[API] chunk1")
	assert.Contains(t, output, "[API] chunk2")
}

func TestStreamWriter_NilWriterPanics(t *testing.T) {
	assert.Panics(t, func() {
		NewStreamWriter(nil)
	}, "NewStreamWriter should panic when given nil writer")
}
