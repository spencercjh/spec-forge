package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{
			name:     "with cause",
			err:      &Error{Code: CodeSystem, Message: "command failed", Cause: errors.New("exit status 1")},
			expected: "[SYSTEM] command failed: exit status 1",
		},
		{
			name:     "without cause",
			err:      &Error{Code: CodeDetect, Message: "no pom.xml found"},
			expected: "[DETECT] no pom.xml found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := New(CodeSystem, "wrapper", cause)
	if !errors.Is(err, cause) {
		t.Error("errors.Is should find the cause through Unwrap()")
	}
}

func TestError_WithContext(t *testing.T) {
	original := New(CodeDetect, "not found", nil)
	updated := original.WithContext("path", "/some/path")

	if len(original.Context) != 0 {
		t.Error("WithContext should not mutate the receiver")
	}
	if updated.Context["path"] != "/some/path" {
		t.Errorf("expected context path = %q, got %q", "/some/path", updated.Context["path"])
	}
	if updated.Code != original.Code || updated.Message != original.Message {
		t.Error("WithContext should preserve Code and Message")
	}
}

func TestError_WithContext_Chaining(t *testing.T) {
	err := New(CodeDetect, "not found", nil).
		WithContext("path", "/some/path").
		WithContext("tool", "mvn").
		WithContext("extra", 42)

	if err.Context["path"] != "/some/path" {
		t.Errorf("expected path = %q, got %q", "/some/path", err.Context["path"])
	}
	if err.Context["tool"] != "mvn" {
		t.Errorf("expected tool = %q, got %q", "mvn", err.Context["tool"])
	}
	if err.Context["extra"] != 42 {
		t.Errorf("expected extra = 42, got %v", err.Context["extra"])
	}
	if err.Code != CodeDetect {
		t.Errorf("chaining should preserve Code, got %q", err.Code)
	}
}

func TestError_Hint(t *testing.T) {
	err := DetectError("no build file", nil)
	hint := err.Hint()
	if hint == "" {
		t.Error("Hint() should return a non-empty string for a known code")
	}
}

func TestNew(t *testing.T) {
	cause := errors.New("cause")
	err := New(CodeConfig, "bad config", cause)
	if err.Code != CodeConfig {
		t.Errorf("Code = %q, want %q", err.Code, CodeConfig)
	}
	if err.Message != "bad config" {
		t.Errorf("Message = %q, want %q", err.Message, "bad config")
	}
	if err.Cause != cause {
		t.Error("Cause should be set")
	}
}

func TestNewf(t *testing.T) {
	cause := errors.New("cause")
	err := Newf(CodeGenerate, cause, "failed %s with code %d", "maven", 1)
	if err.Message != "failed maven with code 1" {
		t.Errorf("Message = %q, want %q", err.Message, "failed maven with code 1")
	}
}

func TestIsCode(t *testing.T) {
	err := SystemError("timeout", nil)
	if !IsCode(err, CodeSystem) {
		t.Error("IsCode should return true for matching code")
	}
	if IsCode(err, CodeDetect) {
		t.Error("IsCode should return false for non-matching code")
	}

	// wrapped error
	wrapped := fmt.Errorf("outer: %w", err)
	if !IsCode(wrapped, CodeSystem) {
		t.Error("IsCode should work through error wrapping")
	}

	// nil
	if IsCode(nil, CodeSystem) {
		t.Error("IsCode should return false for nil error")
	}

	// nested classified errors: outer=DETECT wrapping inner=SYSTEM
	inner := SystemError("timeout", nil)
	outer := DetectError("detect failed", inner)
	if !IsCode(outer, CodeSystem) {
		t.Error("IsCode should find SYSTEM code nested inside DETECT error")
	}
	if !IsCode(outer, CodeDetect) {
		t.Error("IsCode should find DETECT code in outer error")
	}
}

func TestGetCode(t *testing.T) {
	err := LLMError("rate limited", nil)
	if got := GetCode(err); got != CodeLLM {
		t.Errorf("GetCode = %q, want %q", got, CodeLLM)
	}

	// Non-Error error
	plain := errors.New("plain")
	if got := GetCode(plain); got != "" {
		t.Errorf("GetCode(plain) = %q, want empty string", got)
	}

	// nil error
	if got := GetCode(nil); got != "" {
		t.Errorf("GetCode(nil) = %q, want empty string", got)
	}
}

func TestIsRetryable(t *testing.T) {
	retryable := []error{
		LLMError("rate limited", nil),
		PublishError("network error", nil),
		SystemError("timeout", nil),
	}
	for _, err := range retryable {
		if !IsRetryable(err) {
			t.Errorf("IsRetryable(%v) = false, want true", err)
		}
	}

	notRetryable := []error{
		ConfigError("bad config", nil),
		DetectError("not found", nil),
		PatchError("write failed", nil),
		GenerateError("build failed", nil),
		ValidateError("invalid spec", nil),
		errors.New("plain error"),
		nil,
	}
	for _, err := range notRetryable {
		if IsRetryable(err) {
			t.Errorf("IsRetryable(%v) = true, want false", err)
		}
	}
}

func TestConvenienceConstructors(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		wantCode string
	}{
		{"ConfigError", ConfigError("msg", nil), CodeConfig},
		{"DetectError", DetectError("msg", nil), CodeDetect},
		{"PatchError", PatchError("msg", nil), CodePatch},
		{"GenerateError", GenerateError("msg", nil), CodeGenerate},
		{"ValidateError", ValidateError("msg", nil), CodeValidate},
		{"LLMError", LLMError("msg", nil), CodeLLM},
		{"PublishError", PublishError("msg", nil), CodePublish},
		{"SystemError", SystemError("msg", nil), CodeSystem},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", tt.err.Code, tt.wantCode)
			}
		})
	}
}

// TestCrossPackageErrorChain verifies that error classification works when
// errors are wrapped by outer packages using fmt.Errorf("%w", ...).
func TestCrossPackageErrorChain(t *testing.T) {
	inner := SystemError("command timed out", nil)

	// Simulate wrapping by an outer package
	outer := fmt.Errorf("extractor failed: %w", inner)

	if !IsCode(outer, CodeSystem) {
		t.Error("IsCode should find SYSTEM code through fmt.Errorf wrapping")
	}
	if GetCode(outer) != CodeSystem {
		t.Errorf("GetCode = %q, want %q", GetCode(outer), CodeSystem)
	}
	if !IsRetryable(outer) {
		t.Error("IsRetryable should return true for SYSTEM code through wrapping")
	}

	// errors.As should also work through the chain
	var fe *Error
	if !errors.As(outer, &fe) {
		t.Error("errors.As should find *Error through wrapping")
	}
	if fe.Code != CodeSystem {
		t.Errorf("errors.As found code = %q, want %q", fe.Code, CodeSystem)
	}
}
