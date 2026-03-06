// Package gozero provides go-zero framework specific extraction functionality.
package gozero

import (
	"context"
	"fmt"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

// Generator generates OpenAPI specs from go-zero projects.
type Generator struct {
	detector *Detector
}

// NewGenerator creates a new Generator instance.
func NewGenerator() *Generator {
	return &Generator{
		detector: NewDetector(),
	}
}

// Generate generates OpenAPI spec by invoking goctl tool.
func (g *Generator) Generate(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
	// TODO: Implement go-zero OpenAPI spec generation using goctl
	return nil, fmt.Errorf("not implemented: go-zero spec generation")
}
