// Package gozero_test tests the go-zero extractor implementation.
package gozero_test

import (
	"testing"

	"github.com/spencercjh/spec-forge/internal/extractor/gozero"
)

func TestNewDetector(t *testing.T) {
	d := gozero.NewDetector()
	if d == nil {
		t.Error("NewDetector() should not return nil")
	}
}

func TestDetector_Detect_NotImplemented(t *testing.T) {
	d := gozero.NewDetector()
	_, err := d.Detect("/tmp/test-project")

	if err == nil {
		t.Error("Detect should return error for unimplemented method")
	}

	if err != nil && err.Error() != "not implemented: go-zero project detection" {
		t.Errorf("Detect error message = %v, want 'not implemented: go-zero project detection'", err)
	}
}
