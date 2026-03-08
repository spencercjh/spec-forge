// Package grpcprotoc provides gRPC-protoc framework extraction functionality.
package grpcprotoc

import "github.com/spencercjh/spec-forge/internal/extractor"

// BuildToolProtoc represents protoc as the build tool.
const BuildToolProtoc extractor.BuildTool = "protoc"

// Info holds gRPC-protoc specific project information.
type Info struct {
	ProtoFiles        []string // All .proto files found
	ServiceProtoFiles []string // Proto files with service definitions (main entry points)
	ProtoRoot         string   // Root directory containing proto files
	HasGoogleAPI      bool     // Whether google/api/annotations.proto is imported
	HasBuf            bool     // Whether buf.yaml exists (should be false)
	ImportPaths       []string // Detected import paths
}
