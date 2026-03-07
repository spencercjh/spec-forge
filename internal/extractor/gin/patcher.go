package gin

import (
	"context"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

// Patcher is a no-op for Gin projects (no patching needed).
type Patcher struct{}

// NewPatcher creates a new Patcher instance.
func NewPatcher() *Patcher {
	return &Patcher{}
}

// Patch performs no-op patching for Gin projects.
func (p *Patcher) Patch(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.PatchOptions) (*extractor.PatchResult, error) {
	return &extractor.PatchResult{}, nil
}
