// Package grpcprotoc_test tests the gRPC-protoc extractor implementation.
package grpcprotoc_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/spencercjh/spec-forge/internal/extractor/grpcprotoc"
)

func TestNewDetector(t *testing.T) {
	d := grpcprotoc.NewDetector()
	if d == nil {
		t.Error("NewDetector() should not return nil")
	}
}

func TestDetector_Detect_BufProjectRejected(t *testing.T) {
	// Create temp dir with buf.yaml (should be rejected)
	tmpDir := t.TempDir()

	// Create buf.yaml
	bufYamlPath := filepath.Join(tmpDir, "buf.yaml")
	if err := os.WriteFile(bufYamlPath, []byte("version: v1"), 0o644); err != nil {
		t.Fatalf("Failed to create buf.yaml: %v", err)
	}

	// Create a .proto file
	protoFile := filepath.Join(tmpDir, "test.proto")
	protoContent := `syntax = "proto3";
package test;
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0o644); err != nil {
		t.Fatalf("Failed to create .proto file: %v", err)
	}

	detector := grpcprotoc.NewDetector()
	_, err := detector.Detect(tmpDir)

	if err == nil {
		t.Fatal("Expected error when buf.yaml is present")
	}

	if !errors.Is(err, grpcprotoc.ErrBufProjectDetected) {
		t.Errorf("Expected ErrBufProjectDetected, got: %v", err)
	}
}

func TestDetector_Detect_NoProtoFiles(t *testing.T) {
	// Create temp dir without any .proto files
	tmpDir := t.TempDir()

	// Create a readme file (no proto files)
	readmePath := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test Project"), 0o644); err != nil {
		t.Fatalf("Failed to create README.md: %v", err)
	}

	detector := grpcprotoc.NewDetector()
	_, err := detector.Detect(tmpDir)

	if err == nil {
		t.Fatal("Expected error when no .proto files are found")
	}

	if _, ok := errors.AsType[*grpcprotoc.ErrNotProtocProject](err); !ok {
		t.Errorf("Expected error to be *grpcprotoc.ErrNotProtocProject, got: %T", err)
	}
}

func TestDetector_Detect_ValidProtocProject(t *testing.T) {
	// Create temp dir with valid protoc project (proto files, no buf.yaml)
	tmpDir := t.TempDir()

	// Create proto directory
	protoDir := filepath.Join(tmpDir, "proto")
	if err := os.MkdirAll(protoDir, 0o755); err != nil {
		t.Fatalf("Failed to create proto dir: %v", err)
	}

	// Create a .proto file
	protoFile := filepath.Join(protoDir, "test.proto")
	protoContent := `syntax = "proto3";
package test;

message HelloRequest {
  string name = 1;
}

message HelloResponse {
  string message = 1;
}

service HelloService {
  rpc SayHello(HelloRequest) returns (HelloResponse);
}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0o644); err != nil {
		t.Fatalf("Failed to create .proto file: %v", err)
	}

	detector := grpcprotoc.NewDetector()
	info, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if info.Framework != grpcprotoc.ExtractorName {
		t.Errorf("Framework = %s, want %s", info.Framework, grpcprotoc.ExtractorName)
	}

	if info.BuildTool != grpcprotoc.BuildToolProtoc {
		t.Errorf("BuildTool = %s, want %s", info.BuildTool, grpcprotoc.BuildToolProtoc)
	}

	if info.BuildFilePath != tmpDir {
		t.Errorf("BuildFilePath = %s, want %s", info.BuildFilePath, tmpDir)
	}

	if info.FrameworkData == nil {
		t.Fatal("FrameworkData should not be nil")
	}

	grpcInfo, ok := info.FrameworkData.(*grpcprotoc.Info)
	if !ok {
		t.Fatal("FrameworkData should be *grpcprotoc.Info")
	}

	if len(grpcInfo.ProtoFiles) != 1 {
		t.Errorf("Expected 1 .proto file, got %d", len(grpcInfo.ProtoFiles))
	}

	if grpcInfo.HasBuf {
		t.Error("HasBuf should be false for valid protoc project")
	}

	if grpcInfo.HasGoogleAPI {
		t.Error("HasGoogleAPI should be false when no google/api/annotations.proto is imported")
	}

	// Check that import paths include the project root
	foundRoot := false
	for _, path := range grpcInfo.ImportPaths {
		if path == tmpDir || path == protoDir {
			foundRoot = true
			break
		}
	}
	if !foundRoot {
		t.Errorf("ImportPaths should include project root or proto dir, got: %v", grpcInfo.ImportPaths)
	}
}

func TestDetector_Detect_GoogleAPIAnnotations(t *testing.T) {
	// Create temp dir with proto file that imports google/api/annotations.proto
	tmpDir := t.TempDir()

	// Create proto file with google.api.http annotations
	protoFile := filepath.Join(tmpDir, "service.proto")
	protoContent := `syntax = "proto3";
package test;

import "google/api/annotations.proto";

message HelloRequest {
  string name = 1;
}

message HelloResponse {
  string message = 1;
}

service HelloService {
  rpc SayHello(HelloRequest) returns (HelloResponse) {
    option (google.api.http) = {
      get: "/v1/hello"
    };
  }
}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0o644); err != nil {
		t.Fatalf("Failed to create .proto file: %v", err)
	}

	detector := grpcprotoc.NewDetector()
	info, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	grpcInfo, ok := info.FrameworkData.(*grpcprotoc.Info)
	if !ok {
		t.Fatal("FrameworkData should be *grpcprotoc.Info")
	}

	if !grpcInfo.HasGoogleAPI {
		t.Error("HasGoogleAPI should be true when google/api/annotations.proto is imported")
	}
}

func TestDetector_Detect_MultipleProtoFiles(t *testing.T) {
	// Create temp dir with multiple .proto files in different directories
	tmpDir := t.TempDir()

	// Create proto directory
	protoDir := filepath.Join(tmpDir, "proto")
	if err := os.MkdirAll(protoDir, 0o755); err != nil {
		t.Fatalf("Failed to create proto dir: %v", err)
	}

	// Create subdirectory
	subDir := filepath.Join(protoDir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("Failed to create sub dir: %v", err)
	}

	// Create proto files
	proto1 := filepath.Join(protoDir, "service.proto")
	proto1Content := `syntax = "proto3";
package test;

service TestService {
  rpc Get(GetRequest) returns (GetResponse);
}

message GetRequest {}
message GetResponse {}
`
		if err := os.WriteFile(proto1, []byte(proto1Content), 0o644); err != nil {
		t.Fatalf("Failed to create proto1: %v", err)
	}

	proto2 := filepath.Join(subDir, "model.proto")
	if err := os.WriteFile(proto2, []byte(`syntax = "proto3"; package test.model;`), 0o644); err != nil {
		t.Fatalf("Failed to create proto2: %v", err)
	}

	proto3 := filepath.Join(tmpDir, "root.proto")
	if err := os.WriteFile(proto3, []byte(`syntax = "proto3"; package root;`), 0o644); err != nil {
		t.Fatalf("Failed to create proto3: %v", err)
	}

	detector := grpcprotoc.NewDetector()
	info, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	grpcInfo, ok := info.FrameworkData.(*grpcprotoc.Info)
	if !ok {
		t.Fatal("FrameworkData should be *grpcprotoc.Info")
	}

	if len(grpcInfo.ProtoFiles) != 3 {
		t.Errorf("Expected 3 .proto files, got %d: %v", len(grpcInfo.ProtoFiles), grpcInfo.ProtoFiles)
	}
}

func TestDetector_Detect_ImportPaths(t *testing.T) {
	// Create temp dir with various proto directories
	tmpDir := t.TempDir()

	// Create common import path directories
	protoDir := filepath.Join(tmpDir, "proto")
	if err := os.MkdirAll(protoDir, 0o755); err != nil {
		t.Fatalf("Failed to create proto dir: %v", err)
	}

	thirdPartyDir := filepath.Join(tmpDir, "third_party")
	if err := os.MkdirAll(thirdPartyDir, 0o755); err != nil {
		t.Fatalf("Failed to create third_party dir: %v", err)
	}

	protosDir := filepath.Join(tmpDir, "protos")
	if err := os.MkdirAll(protosDir, 0o755); err != nil {
		t.Fatalf("Failed to create protos dir: %v", err)
	}

	// Create proto files in each directory (with service definition in first one)
	aProtoContent := `syntax = "proto3";

service TestService {}
`
		if err := os.WriteFile(filepath.Join(protoDir, "a.proto"), []byte(aProtoContent), 0o644); err != nil {
		t.Fatalf("Failed to create a.proto: %v", err)
	}
	if err := os.WriteFile(filepath.Join(thirdPartyDir, "b.proto"), []byte(`syntax = "proto3";`), 0o644); err != nil {
		t.Fatalf("Failed to create b.proto: %v", err)
	}
	if err := os.WriteFile(filepath.Join(protosDir, "c.proto"), []byte(`syntax = "proto3";`), 0o644); err != nil {
		t.Fatalf("Failed to create c.proto: %v", err)
	}

	detector := grpcprotoc.NewDetector()
	info, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	grpcInfo, ok := info.FrameworkData.(*grpcprotoc.Info)
	if !ok {
		t.Fatal("FrameworkData should be *grpcprotoc.Info")
	}

	// Check that import paths include proto, third_party, and protos directories
	expectedPaths := []string{protoDir, thirdPartyDir, protosDir}
	for _, expected := range expectedPaths {
		if !slices.Contains(grpcInfo.ImportPaths, expected) {
			t.Errorf("ImportPaths should include %s, got: %v", expected, grpcInfo.ImportPaths)
		}
	}
}

func TestDetector_Detect_SkipVendor(t *testing.T) {
	// Create temp dir with vendor directory
	tmpDir := t.TempDir()

	// Create proto file in main directory with service definition
	protoFile := filepath.Join(tmpDir, "main.proto")
	if err := os.WriteFile(protoFile, []byte(`syntax = "proto3";
package main;
service MainService { rpc GetData(GetRequest) returns (GetResponse); }
message GetRequest {}
message GetResponse {}`), 0o644); err != nil {
		t.Fatalf("Failed to create main.proto: %v", err)
	}

	// Create vendor directory with proto file (should be skipped)
	vendorDir := filepath.Join(tmpDir, "vendor", "github.com", "example")
	if err := os.MkdirAll(vendorDir, 0o755); err != nil {
		t.Fatalf("Failed to create vendor dir: %v", err)
	}

	vendorProto := filepath.Join(vendorDir, "vendor.proto")
	if err := os.WriteFile(vendorProto, []byte(`syntax = "proto3";`), 0o644); err != nil {
		t.Fatalf("Failed to create vendor.proto: %v", err)
	}

	detector := grpcprotoc.NewDetector()
	info, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	grpcInfo, ok := info.FrameworkData.(*grpcprotoc.Info)
	if !ok {
		t.Fatal("FrameworkData should be *grpcprotoc.Info")
	}

	// Should only find the main.proto file, not the vendor one
	if len(grpcInfo.ProtoFiles) != 1 {
		t.Errorf("Expected 1 .proto file (vendor skipped), got %d: %v", len(grpcInfo.ProtoFiles), grpcInfo.ProtoFiles)
	}

	if len(grpcInfo.ProtoFiles) > 0 && grpcInfo.ProtoFiles[0] != protoFile {
		t.Errorf("Expected %s, got %s", protoFile, grpcInfo.ProtoFiles[0])
	}
}

func TestDetector_Detect_SkipHiddenDirs(t *testing.T) {
	// Create temp dir with hidden directory
	tmpDir := t.TempDir()

	// Create proto file in main directory with service definition
	protoFile := filepath.Join(tmpDir, "main.proto")
	if err := os.WriteFile(protoFile, []byte(`syntax = "proto3";
package main;
service MainService { rpc GetData(GetRequest) returns (GetResponse); }
message GetRequest {}
message GetResponse {}`), 0o644); err != nil {
		t.Fatalf("Failed to create main.proto: %v", err)
	}

	// Create hidden directory with proto file (should be skipped)
	hiddenDir := filepath.Join(tmpDir, ".git", "hooks")
	if err := os.MkdirAll(hiddenDir, 0o755); err != nil {
		t.Fatalf("Failed to create hidden dir: %v", err)
	}

	hiddenProto := filepath.Join(hiddenDir, "hidden.proto")
	if err := os.WriteFile(hiddenProto, []byte(`syntax = "proto3";`), 0o644); err != nil {
		t.Fatalf("Failed to create hidden.proto: %v", err)
	}

	detector := grpcprotoc.NewDetector()
	info, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	grpcInfo, ok := info.FrameworkData.(*grpcprotoc.Info)
	if !ok {
		t.Fatal("FrameworkData should be *grpcprotoc.Info")
	}

	// Should only find the main.proto file, not the hidden one
	if len(grpcInfo.ProtoFiles) != 1 {
		t.Errorf("Expected 1 .proto file (hidden skipped), got %d", len(grpcInfo.ProtoFiles))
	}
}

func TestDetector_Detect_InvalidPath(t *testing.T) {
	detector := grpcprotoc.NewDetector()
	_, err := detector.Detect("/nonexistent/path/that/does/not/exist")

	if err == nil {
		t.Error("Expected error for invalid path")
	}
}
