package cli

import (
	"bytes"
	"os"
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
)

func TestColorEnabled(t *testing.T) {
	t.Run("NO_COLOR=1 disables color", func(t *testing.T) {
		os.Setenv("NO_COLOR", "1")
		defer os.Unsetenv("NO_COLOR")
		origColorNoColor := color.NoColor
		defer func() { color.NoColor = origColorNoColor }()

		initColorState()

		assert.True(t, color.NoColor)
		assert.False(t, ColorEnabled())
	})

	t.Run("NO_COLOR empty string disables color", func(t *testing.T) {
		os.Setenv("NO_COLOR", "")
		defer os.Unsetenv("NO_COLOR")
		origColorNoColor := color.NoColor
		defer func() { color.NoColor = origColorNoColor }()

		initColorState()

		assert.True(t, color.NoColor)
		assert.False(t, ColorEnabled())
	})

	t.Run("no NO_COLOR preserves existing color.NoColor", func(t *testing.T) {
		os.Unsetenv("NO_COLOR")
		origColorNoColor := color.NoColor
		defer func() { color.NoColor = origColorNoColor }()

		// Verify invariant: absent NO_COLOR, initColorState must not override color.NoColor
		for _, initial := range []bool{false, true} {
			color.NoColor = initial
			initColorState()
			assert.Equal(t, initial, color.NoColor, "initColorState should not override color.NoColor when NO_COLOR is absent")
		}
	})
}

func TestStatusFunctions(t *testing.T) {
	origNoColor := os.Getenv("NO_COLOR")
	os.Unsetenv("NO_COLOR")
	defer os.Setenv("NO_COLOR", origNoColor)
	origColorNoColor := color.NoColor
	defer func() { color.NoColor = origColorNoColor }()
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
	origColorNoColor := color.NoColor
	defer func() { color.NoColor = origColorNoColor }()
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
