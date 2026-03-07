// Package gin provides Gin framework specific extraction functionality.
package gin

import (
	"context"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

const FrameworkName = "gin"

// GinExtractor implements extractor.Extractor for Gin projects.
type GinExtractor struct {
	detector  *Detector
	patcher   *Patcher
	generator *Generator
}

// NewGinExtractor creates a new GinExtractor instance.
func NewGinExtractor() *GinExtractor {
	return &GinExtractor{
		detector:  NewDetector(),
		patcher:   NewPatcher(),
		generator: NewGenerator(),
	}
}

// Detect implements extractor.Extractor.Detect.
func (e *GinExtractor) Detect(projectPath string) (*extractor.ProjectInfo, error) {
	return e.detector.Detect(projectPath)
}

// Patch implements extractor.Extractor.Patch.
func (e *GinExtractor) Patch(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.PatchOptions) (*extractor.PatchResult, error) {
	return e.patcher.Patch(ctx, projectPath, info, opts)
}

// Generate implements extractor.Extractor.Generate.
func (e *GinExtractor) Generate(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
	return e.generator.Generate(ctx, projectPath, info, opts)
}
