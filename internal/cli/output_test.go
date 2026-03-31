package cli

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestColorEnabled(t *testing.T) {
	t.Run("NO_COLOR=1 disables color", func(t *testing.T) {
		os.Setenv("NO_COLOR", "1")
		defer os.Unsetenv("NO_COLOR")

		initColorState()
		assert.False(t, ColorEnabled())
	})

	t.Run("NO_COLOR empty string disables color", func(t *testing.T) {
		os.Setenv("NO_COLOR", "")
		defer os.Unsetenv("NO_COLOR")

		initColorState()
		assert.False(t, ColorEnabled())
	})

	t.Run("no NO_COLOR delegates to fatih/color TTY detection", func(t *testing.T) {
		os.Unsetenv("NO_COLOR")

		initColorState()
		// In non-TTY test environments, fatih/color sets NoColor=true by default.
		// We verify the delegation works rather than asserting a specific value.
		// The key invariant: NO_COLOR absent means we do NOT force color.NoColor.
		assert.False(t, ColorEnabled()) // non-TTY in test
	})
}

func TestStatusFunctions(t *testing.T) {
	origNoColor := os.Getenv("NO_COLOR")
	os.Unsetenv("NO_COLOR")
	defer os.Setenv("NO_COLOR", origNoColor)
	initColorState()

	t.Run("Successf contains checkmark prefix", func(t *testing.T) {
		var buf bytes.Buffer
		Successf(&buf, "test message")
		output := buf.String()
		assert.Contains(t, output, "✅")
		assert.Contains(t, output, "test message")
	})

	t.Run("Skipf contains skip prefix", func(t *testing.T) {
		var buf bytes.Buffer
		Skipf(&buf, "skipped")
		output := buf.String()
		assert.Contains(t, output, "⏭️")
		assert.Contains(t, output, "skipped")
	})

	t.Run("Errorf contains error prefix", func(t *testing.T) {
		var buf bytes.Buffer
		Errorf(&buf, "something failed")
		output := buf.String()
		assert.Contains(t, output, "❌")
		assert.Contains(t, output, "something failed")
	})

	t.Run("Hintf contains hint prefix", func(t *testing.T) {
		var buf bytes.Buffer
		Hintf(&buf, "check your config")
		output := buf.String()
		assert.Contains(t, output, "💡 Hint:")
		assert.Contains(t, output, "check your config")
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
		assert.Contains(t, buf.String(), "❌")
	})

	t.Run("Hintf no ANSI codes", func(t *testing.T) {
		var buf bytes.Buffer
		Hintf(&buf, "test")
		assert.NotContains(t, buf.String(), "\x1b[")
		assert.Contains(t, buf.String(), "💡 Hint:")
	})
}
