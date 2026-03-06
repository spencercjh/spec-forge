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
	if info.Spring != nil {
		t.Error("Spring should default to nil")
	}
	if info.GoZero != nil {
		t.Error("GoZero should default to nil")
	}
}

func TestSpringInfoDefaults(t *testing.T) {
	info := extractor.SpringInfo{}
	if info.HasSpringdocDeps {
		t.Error("HasSpringdocDeps should default to false")
	}
	if info.HasSpringdocPlugin {
		t.Error("HasSpringdocPlugin should default to false")
	}
}

func TestGoZeroInfoDefaults(t *testing.T) {
	info := extractor.GoZeroInfo{}
	if info.HasGoZeroDeps {
		t.Error("HasGoZeroDeps should default to false")
	}
	if info.HasGoctl {
		t.Error("HasGoctl should default to false")
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
