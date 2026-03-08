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
func (p *Patcher) Patch(_ context.Context, _ string, info *extractor.ProjectInfo, _ *extractor.PatchOptions) (*extractor.PatchResult, error) {
	// Gin projects don't need patching, just mark as ready
	if ginInfo, ok := info.FrameworkData.(*Info); ok {
		ginInfo.HasGin = true
	}
	return &extractor.PatchResult{}, nil
}
