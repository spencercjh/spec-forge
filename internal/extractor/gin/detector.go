package gin

import "github.com/spencercjh/spec-forge/internal/extractor"

// Detector detects Gin projects.
type Detector struct{}

// NewDetector creates a new Detector instance.
func NewDetector() *Detector {
	return &Detector{}
}

// Detect analyzes a project and returns info if it's a Gin project.
func (d *Detector) Detect(projectPath string) (*extractor.ProjectInfo, error) {
	return nil, nil
}
