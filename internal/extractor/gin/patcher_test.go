package gin

import (
	"context"
	"testing"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

func TestNewPatcher(t *testing.T) {
	p := NewPatcher()
	if p == nil {
		t.Error("expected non-nil patcher")
	}
}

func TestPatcher_Patch(t *testing.T) {
	p := NewPatcher()
	ctx := context.Background()
	info := &extractor.ProjectInfo{}
	opts := &extractor.PatchOptions{}

	result, err := p.Patch(ctx, "/tmp", info, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Patcher should return empty result for Gin
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestPatcher_Patch_SetsFrameworkData(t *testing.T) {
	p := NewPatcher()
	ctx := context.Background()
	ginInfo := &Info{HasGin: false}
	info := &extractor.ProjectInfo{
		Framework:     ExtractorName,
		FrameworkData: ginInfo,
	}
	opts := &extractor.PatchOptions{}

	_, err := p.Patch(ctx, "/tmp", info, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Gin projects don't need patching, just mark as ready
	if !ginInfo.HasGin {
		t.Error("expected HasGin to be true after patch")
	}
}
