// Package grpcprotoc provides gRPC-protoc framework extraction functionality.
package grpcprotoc

import (
	"context"

	forgeerrors "github.com/spencercjh/spec-forge/internal/errors"
	"github.com/spencercjh/spec-forge/internal/extractor"
)

// ExtractorName is the name of this extractor.
const ExtractorName = "grpc-protoc"

// Extractor implements the extractor.Extractor interface for gRPC-protoc projects.
type Extractor struct {
	detector  *Detector
	patcher   *Patcher
	generator *Generator
}

func NewExtractor() *Extractor {
	return &Extractor{
		detector:  NewDetector(),
		patcher:   NewPatcher(),
		generator: NewGenerator(),
	}
}

// Name returns the extractor name.
func (e *Extractor) Name() string {
	return ExtractorName
}

// Detect analyzes a project and returns its information if it's a gRPC-protoc project.
func (e *Extractor) Detect(projectPath string) (*extractor.ProjectInfo, error) {
	info, err := e.detector.Detect(projectPath)
	if err != nil {
		// Detector already returns DETECT-classified errors; avoid double-wrapping
		return nil, err
	}
	return info, nil
}

// Patch checks if protoc and protoc-gen-connect-openapi are available for the gRPC-protoc project.
func (e *Extractor) Patch(_ string, _ *extractor.PatchOptions) (*extractor.PatchResult, error) {
	_, err := e.patcher.Patch("")
	if err != nil {
		// Patcher already returns PATCH-classified errors; avoid double-wrapping
		return nil, err
	}
	// gRPC-protoc doesn't modify project files, so return empty result.
	return &extractor.PatchResult{}, nil
}

// Generate produces the OpenAPI spec from the gRPC-protoc project.
func (e *Extractor) Generate(ctx context.Context, projectPath string, info *extractor.ProjectInfo, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
	result, err := e.generator.Generate(ctx, projectPath, info, opts)
	if err != nil {
		return nil, forgeerrors.GenerateError("grpc-protoc spec generation failed", err)
	}
	return result, nil
}

// Restore is a no-op for gRPC-protoc projects as we don't modify files.
func (e *Extractor) Restore(_, _ string) error {
	return nil
}
