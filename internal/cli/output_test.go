package cli

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestColorEnabled(t *testing.T) {
	t.Run("color enabled by default", func(t *testing.T) {
		orig := os.Getenv("NO_COLOR")
		os.Unsetenv("NO_COLOR")
		defer os.Setenv("NO_COLOR", orig)

		initColorState()
		assert.True(t, ColorEnabled())
	})

	t.Run("color disabled with NO_COLOR", func(t *testing.T) {
		os.Setenv("NO_COLOR", "1")
		defer os.Unsetenv("NO_COLOR")

		initColorState()
		assert.False(t, ColorEnabled())
	})
}

func TestStatusFunctions(t *testing.T) {
	origNoColor := os.Getenv("NO_COLOR")
	os.Unsetenv("NO_COLOR")
	defer os.Setenv("NO_COLOR", origNoColor)
	initColorState()

	t.Run("Successf contains checkmark", func(t *testing.T) {
		var buf bytes.Buffer
		Successf(&buf, "test message")
		assert.Contains(t, buf.String(), "test message")
		assert.Contains(t, buf.String(), "✅")
	})

	t.Run("Skipf contains skip mark", func(t *testing.T) {
		var buf bytes.Buffer
		Skipf(&buf, "skipped")
		assert.Contains(t, buf.String(), "skipped")
		assert.Contains(t, buf.String(), "⏭️")
	})

	t.Run("Errorf contains error marker", func(t *testing.T) {
		var buf bytes.Buffer
		Errorf(&buf, "something failed")
		assert.Contains(t, buf.String(), "something failed")
	})

	t.Run("Hintf contains hint", func(t *testing.T) {
		var buf bytes.Buffer
		Hintf(&buf, "check your config")
		assert.Contains(t, buf.String(), "check your config")
	})

	t.Run("Statusf formats with args", func(t *testing.T) {
		var buf bytes.Buffer
		Statusf(&buf, "found %d items", 5)
		assert.Contains(t, buf.String(), "found 5 items")
	})
}

func TestNoColorMode(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")
	initColorState()

	t.Run("Successf no ANSI codes", func(t *testing.T) {
		var buf bytes.Buffer
		Successf(&buf, "test")
		assert.NotContains(t, buf.String(), "\x1b[")
		assert.Contains(t, buf.String(), "✅")
	})

	t.Run("Errorf no ANSI codes", func(t *testing.T) {
		var buf bytes.Buffer
		Errorf(&buf, "test")
		assert.NotContains(t, buf.String(), "\x1b[")
	})
}
