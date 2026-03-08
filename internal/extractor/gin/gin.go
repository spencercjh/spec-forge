// Package gin provides Gin framework specific extraction functionality.
package gin

import (
	"context"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

// ExtractorName is the name of the Gin extractor.
const ExtractorName = "gin"

// Extractor implements extractor.Extractor for Gin projects.
type Extractor struct {
	detector  *Detector
	patcher   *Patcher
	generator *Generator
}

// ensureInitialized lazily initializes the extractor fields if needed.
// This handles the case where the extractor is used as a zero-value.
func (e *Extractor) ensureInitialized() {
	if e.detector == nil {
		e.detector = NewDetector()
	}
	if e.patcher == nil {
		e.patcher = NewPatcher()
	}
	if e.generator == nil {
		e.generator = NewGenerator()
	}
}

// NewExtractor creates a new Extractor instance.
func NewExtractor() *Extractor {
	return &Extractor{
		detector:  NewDetector(),
		patcher:   NewPatcher(),
		generator: NewGenerator(),
	}
}

// Name returns the extractor name.
func (e *Extractor) Name() string {
	return ExtractorName
}

// Detect implements extractor.Extractor.Detect.
func (e *Extractor) Detect(projectPath string) (*extractor.ProjectInfo, error) {
	e.ensureInitialized()
	return e.detector.Detect(projectPath)
}

// Patch implements extractor.Extractor.Patch.
func (e *Extractor) Patch(projectPath string, opts *extractor.PatchOptions) (*extractor.PatchResult, error) {
	e.ensureInitialized()
	// Gin projects don't need patching
	info := &extractor.ProjectInfo{
		Framework:     ExtractorName,
		BuildTool:     "gomodules",
		FrameworkData: &Info{HasGin: true},
	}
	return e.patcher.Patch(context.Background(), projectPath, info, opts)
}

// Generate implements extractor.Extractor.Generate.
func (e *Extractor) Generate(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
	e.ensureInitialized()
	return e.generator.Generate(ctx, projectPath, info, opts)
}

// Restore implements extractor.Extractor.Restore.
func (e *Extractor) Restore(_, _ string) error {
	// Gin projects don't need restore (no patching)
	return nil
}
