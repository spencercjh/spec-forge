// Package gozero_test tests the go-zero extractor implementation.
package gozero_test

import (
	"context"
	"testing"

	"github.com/spencercjh/spec-forge/internal/extractor/gozero"
)

func TestNewGenerator(t *testing.T) {
	g := gozero.NewGenerator()
	if g == nil {
		t.Error("NewGenerator() should not return nil")
	}
}

func TestGenerator_Generate_NotImplemented(t *testing.T) {
	g := gozero.NewGenerator()
	ctx := context.Background()
	_, err := g.Generate(ctx, "/tmp/test-project", nil, nil)

	if err == nil {
		t.Error("Generate should return error for unimplemented method")
	}

	if err != nil && err.Error() != "not implemented: go-zero spec generation" {
		t.Errorf("Generate error message = %v, want 'not implemented: go-zero spec generation'", err)
	}
}
