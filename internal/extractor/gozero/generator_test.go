// Package gozero_test tests the go-zero extractor implementation.
package gozero_test

import (
	"context"
	"testing"

	"github.com/spencercjh/spec-forge/internal/extractor"
	"github.com/spencercjh/spec-forge/internal/extractor/gozero"
)

func TestNewGenerator(t *testing.T) {
	g := gozero.NewGenerator()
	if g == nil {
		t.Error("NewGenerator() should not return nil")
	}
}

func TestGenerator_Generate_NoGoMod(t *testing.T) {
	g := gozero.NewGenerator()
	ctx := context.Background()
	_, err := g.Generate(ctx, "/tmp/non-existent-project", &extractor.ProjectInfo{}, &extractor.GenerateOptions{})

	if err == nil {
		t.Error("Generate should return error for project without go.mod")
	}
}
