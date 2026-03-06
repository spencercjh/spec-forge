// Package gozero provides go-zero framework specific extraction functionality.
package gozero

import (
	"fmt"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

// Detector detects go-zero project information.
type Detector struct {
}

// NewDetector creates a new Detector instance.
func NewDetector() *Detector {
	return &Detector{}
}

// Detect analyzes a go-zero project and returns its information.
func (d *Detector) Detect(projectPath string) (*extractor.ProjectInfo, error) {
	// TODO: Implement go-zero project detection
	// This should check for go.mod file and go-zero dependencies
	return nil, fmt.Errorf("not implemented: go-zero project detection")
}
