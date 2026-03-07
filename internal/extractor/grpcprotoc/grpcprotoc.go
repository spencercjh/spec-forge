// Package grpcprotoc provides gRPC-protoc framework extraction functionality.
package grpcprotoc

// FrameworkName is the identifier for this extractor.
const FrameworkName = "grpc-protoc"

// Info holds gRPC-protoc specific project information.
type Info struct {
	ProtoFiles   []string // All .proto files found
	ProtoRoot    string   // Root directory containing proto files
	HasGoogleAPI bool     // Whether google/api/annotations.proto is imported
	HasBuf       bool     // Whether buf.yaml exists (should be false)
	ImportPaths  []string // Detected import paths
}
