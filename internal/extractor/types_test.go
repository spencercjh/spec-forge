// Package extractor_test tests the extractor types and interfaces.
package extractor_test

import (
	"testing"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

func TestProjectInfoDefaults(t *testing.T) {
	info := extractor.ProjectInfo{}
	if info.BuildTool != "" {
		t.Error("BuildTool should default to empty")
	}
	if info.Framework != "" {
		t.Error("Framework should default to empty")
	}
	if info.FrameworkData != nil {
		t.Error("FrameworkData should default to nil")
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
