// Package gozero_test tests the go-zero extractor implementation.
package gozero_test

import (
	"testing"

	"github.com/spencercjh/spec-forge/internal/extractor/gozero"
)

func TestNewPatcher(t *testing.T) {
	p := gozero.NewPatcher()
	if p == nil {
		t.Error("NewPatcher() should not return nil")
	}
}

func TestPatcher_Patch_NotImplemented(t *testing.T) {
	p := gozero.NewPatcher()
	err := p.Patch("/tmp/test-project", nil)

	if err == nil {
		t.Error("Patch should return error for unimplemented method")
	}

	if err != nil && err.Error() != "not implemented: go-zero project patching" {
		t.Errorf("Patch error message = %v, want 'not implemented: go-zero project patching'", err)
	}
}

func TestPatcher_Restore_NotImplemented(t *testing.T) {
	p := gozero.NewPatcher()
	err := p.Restore("/tmp/test-project")

	if err == nil {
		t.Error("Restore should return error for unimplemented method")
	}

	if err != nil && err.Error() != "not implemented: go-zero project restore" {
		t.Errorf("Restore error message = %v, want 'not implemented: go-zero project restore'", err)
	}
}
