// Package gozero provides go-zero framework specific extraction functionality.
package gozero

import (
	"fmt"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

// Patcher patches go-zero projects for OpenAPI generation.
type Patcher struct {
}

// NewPatcher creates a new Patcher instance.
func NewPatcher() *Patcher {
	return &Patcher{}
}

// Patch modifies the project to enable OpenAPI generation.
func (p *Patcher) Patch(projectPath string, opts *extractor.PatchOptions) error {
	// TODO: Implement go-zero project patching for goctl compatibility
	return fmt.Errorf("not implemented: go-zero project patching")
}

// Restore restores the project to its original state.
func (p *Patcher) Restore(projectPath string) error {
	// TODO: Implement restoration of patched files
	return fmt.Errorf("not implemented: go-zero project restore")
}
