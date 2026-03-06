// Package gozero_test tests the go-zero extractor implementation.
package gozero_test

import (
	"testing"

	"github.com/spencercjh/spec-forge/internal/extractor/gozero"
)

func TestBuildToolGoModules(t *testing.T) {
	if gozero.BuildToolGoModules != "gomodules" {
		t.Errorf("BuildToolGoModules = %s, want gomodules", gozero.BuildToolGoModules)
	}
}

func TestDefaultGoctlVersion(t *testing.T) {
	if gozero.DefaultGoctlVersion == "" {
		t.Error("DefaultGoctlVersion should not be empty")
	}
}

func TestGoctlPackage(t *testing.T) {
	if gozero.GoctlPackage == "" {
		t.Error("GoctlPackage should not be empty")
	}
}
