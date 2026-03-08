package grpcprotoc

import (
	"testing"
)

func TestExtractorName(t *testing.T) {
	if ExtractorName != "grpc-protoc" {
		t.Errorf("ExtractorName = %q, want %q", ExtractorName, "grpc-protoc")
	}
}

func TestInfo(t *testing.T) {
	info := Info{
		ProtoFiles:   []string{"test.proto"},
		ProtoRoot:    "/proto",
		HasGoogleAPI: true,
		HasBuf:       false,
		ImportPaths:  []string{"/proto", "/usr/local/include"},
	}

	if len(info.ProtoFiles) != 1 {
		t.Errorf("len(ProtoFiles) = %d, want 1", len(info.ProtoFiles))
	}

	if info.ProtoRoot != "/proto" {
		t.Errorf("ProtoRoot = %q, want %q", info.ProtoRoot, "/proto")
	}

	if !info.HasGoogleAPI {
		t.Error("HasGoogleAPI should be true")
	}

	if info.HasBuf {
		t.Error("HasBuf should be false")
	}

	if len(info.ImportPaths) != 2 {
		t.Errorf("len(ImportPaths) = %d, want 2", len(info.ImportPaths))
	}
}
