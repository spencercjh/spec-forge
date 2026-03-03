// Package extractor_test tests the extractor types and interfaces.
package extractor_test

import (
	"testing"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

func TestBuildToolConstants(t *testing.T) {
	tests := []struct {
		name     string
		tool     extractor.BuildTool
		expected string
	}{
		{"maven", extractor.BuildToolMaven, "maven"},
		{"gradle", extractor.BuildToolGradle, "gradle"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.tool) != tt.expected {
				t.Errorf("BuildTool %s = %s, want %s", tt.name, tt.tool, tt.expected)
			}
		})
	}
}

func TestDefaultVersions(t *testing.T) {
	if extractor.DefaultSpringdocVersion == "" {
		t.Error("DefaultSpringdocVersion should not be empty")
	}
	if extractor.DefaultSpringdocMavenPlugin == "" {
		t.Error("DefaultSpringdocMavenPlugin should not be empty")
	}
	if extractor.DefaultSpringdocGradlePlugin == "" {
		t.Error("DefaultSpringdocGradlePlugin should not be empty")
	}
}

func TestProjectInfoDefaults(t *testing.T) {
	info := extractor.ProjectInfo{}
	if info.BuildTool != "" {
		t.Error("BuildTool should default to empty")
	}
	if info.HasSpringdocDeps {
		t.Error("HasSpringdocDeps should default to false")
	}
}

func TestPatchOptionsDefaults(t *testing.T) {
	opts := extractor.PatchOptions{}
	if opts.DryRun {
		t.Error("DryRun should default to false")
	}
	if opts.Force {
		t.Error("Force should default to false")
	}
}
