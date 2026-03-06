// Package builtin provides the extractor registry for built-in framework extractors.
// It handles automatic registration and discovery of extractors.
package builtin

import (
	"errors"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

// registry holds all registered extractors.
var registry = make(map[string]extractor.Extractor)

// Register registers an extractor for a framework.
// This should be called in the init() function of each framework package.
func Register(name string, e extractor.Extractor) {
	registry[name] = e
}

// Get returns a registered extractor by name.
func Get(name string) (extractor.Extractor, bool) {
	e, ok := registry[name]
	return e, ok
}

// GetAll returns all registered extractors.
func GetAll() []extractor.Extractor {
	extractors := make([]extractor.Extractor, 0, len(registry))
	for _, e := range registry {
		extractors = append(extractors, e)
	}
	return extractors
}

// DetectFramework tries all registered extractors and returns the first matching framework.
func DetectFramework(projectPath string) (extractor.Extractor, *extractor.ProjectInfo, error) {
	for _, e := range GetAll() {
		info, err := e.Detect(projectPath)
		if err == nil {
			return e, info, nil
		}
	}
	return nil, nil, ErrNoFrameworkDetected
}

// ErrNoFrameworkDetected is returned when no framework can detect the project.
var ErrNoFrameworkDetected = errors.New("no supported framework detected in project")
