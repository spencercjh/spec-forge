// Package gozero_test tests the go-zero extractor implementation.
package gozero_test

import (
	"context"
	"testing"

	"github.com/spencercjh/spec-forge/internal/executor"
	"github.com/spencercjh/spec-forge/internal/extractor/gozero"
)

// mockExecutor is a mock implementation of executor.Interface for testing.
type mockExecutor struct {
	result *executor.ExecuteResult
	err    error
}

func (m *mockExecutor) Execute(ctx context.Context, opts *executor.ExecuteOptions) (*executor.ExecuteResult, error) {
	return m.result, m.err
}

func TestPatcher_Patch_GoctlAvailable(t *testing.T) {
	mockExec := &mockExecutor{
		result: &executor.ExecuteResult{
			ExitCode: 0,
			Stdout:   "goctl version 1.6.0",
			Stderr:   "",
		},
		err: nil,
	}

	patcher := gozero.NewPatcherWithExecutor(mockExec)
	result, err := patcher.Patch("/some/path")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if !result.GoctlAvailable {
		t.Error("Expected GoctlAvailable to be true")
	}

	if result.GoctlVersion != "goctl version 1.6.0" {
		t.Errorf("Expected version 'goctl version 1.6.0', got: %s", result.GoctlVersion)
	}
}

func TestPatcher_Patch_GoctlNotFound(t *testing.T) {
	mockExec := &mockExecutor{
		result: &executor.ExecuteResult{
			ExitCode: -1,
			Stdout:   "",
			Stderr:   "",
		},
		err: &executor.CommandNotFoundError{
			Command: "goctl",
			Hint:    "",
		},
	}

	patcher := gozero.NewPatcherWithExecutor(mockExec)
	result, err := patcher.Patch("/some/path")

	if err == nil {
		t.Error("Expected error when goctl not found, got nil")
	}

	if result != nil {
		t.Error("Expected nil result when goctl not found")
	}

	expectedMsg := "goctl is not installed"
	if err != nil && !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedMsg, err.Error())
	}

	// Verify the installation hint is included
	installHint := "go install github.com/zeromicro/go-zero/tools/goctl@latest"
	if err != nil && !contains(err.Error(), installHint) {
		t.Errorf("Expected error message to contain installation hint '%s', got: %s", installHint, err.Error())
	}
}

func TestPatcher_Patch_GoctlCommandFailed(t *testing.T) {
	mockExec := &mockExecutor{
		result: &executor.ExecuteResult{
			ExitCode: 1,
			Stdout:   "",
			Stderr:   "some error output",
		},
		err: &executor.CommandFailedError{
			Command:  "goctl",
			Args:     []string{"--version"},
			ExitCode: 1,
			Stderr:   "some error output",
		},
	}

	patcher := gozero.NewPatcherWithExecutor(mockExec)
	result, err := patcher.Patch("/some/path")

	if err == nil {
		t.Error("Expected error when goctl command fails, got nil")
	}

	if result != nil {
		t.Error("Expected nil result when goctl command fails")
	}

	expectedMsg := "goctl version check failed"
	if err != nil && !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestPatcher_NeedsPatch(t *testing.T) {
	patcher := gozero.NewPatcher()

	// NeedsPatch should always return true for go-zero projects
	info := &gozero.ProjectInfo{
		HasGoctl: true,
	}

	if !patcher.NeedsPatch(info) {
		t.Error("Expected NeedsPatch to return true")
	}

	info.HasGoctl = false
	if !patcher.NeedsPatch(info) {
		t.Error("Expected NeedsPatch to return true even when HasGoctl is false")
	}
}

func TestNewPatcher(t *testing.T) {
	patcher := gozero.NewPatcher()
	if patcher == nil {
		t.Fatal("Expected patcher to not be nil")
	}
}

func TestNewPatcherWithExecutor(t *testing.T) {
	mockExec := &mockExecutor{
		result: &executor.ExecuteResult{
			ExitCode: 0,
			Stdout:   "goctl version 1.6.0",
		},
		err: nil,
	}

	patcher := gozero.NewPatcherWithExecutor(mockExec)
	if patcher == nil {
		t.Fatal("Expected patcher to not be nil")
	}

	// Verify the custom executor is used by calling Patch
	result, err := patcher.Patch("/some/path")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result == nil || !result.GoctlAvailable {
		t.Error("Expected goctl to be available")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s != "" && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
