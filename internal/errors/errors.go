package errors

import (
	"errors"
	"fmt"
	"maps"
)

// Error is the unified error type for spec-forge. It carries a category code,
// a human-readable message, an optional underlying cause, and optional key/value
// context for debugging.
type Error struct {
	// Code is one of the error category constants (CONFIG, DETECT, PATCH, …).
	Code string
	// Message is a human-readable description of what went wrong.
	Message string
	// Cause is the underlying error that triggered this error (may be nil).
	Cause error
	// Context holds additional key/value pairs for debugging.
	Context map[string]any
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause so that errors.Is / errors.As work through
// the chain.
func (e *Error) Unwrap() error {
	return e.Cause
}

// WithContext returns a shallow copy of the error with an additional key/value
// pair stored in the Context map. It never mutates the receiver.
func (e *Error) WithContext(key string, value any) *Error {
	newCtx := make(map[string]any, len(e.Context)+1)
	maps.Copy(newCtx, e.Context)
	newCtx[key] = value
	return &Error{
		Code:    e.Code,
		Message: e.Message,
		Cause:   e.Cause,
		Context: newCtx,
	}
}

// Hint returns the recovery hint for this error's category code.
func (e *Error) Hint() string {
	return RecoveryHint(e.Code)
}

// New creates a new Error with the given code, message, and cause.
func New(code, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Newf creates a new Error with the given code, cause, and a formatted message.
func Newf(code string, cause error, format string, args ...any) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   cause,
	}
}

// IsCode reports whether any error in err's chain has the given category code.
// It traverses through nested classified errors to find matching codes at any level.
func IsCode(err error, code string) bool {
	for e := err; e != nil; {
		if fe, ok := errors.AsType[*Error](e); ok {
			if fe.Code == code {
				return true
			}
			// Continue checking the Cause if it's also a classified error
			e = fe.Cause
			continue
		}
		e = errors.Unwrap(e)
	}
	return false
}

// GetCode returns the category code of the first *Error found in err's chain.
// It returns an empty string if no *Error is found.
func GetCode(err error) string {
	if fe, ok := errors.AsType[*Error](err); ok {
		return fe.Code
	}
	return ""
}

// IsRetryable reports whether the error is considered retryable based on its
// category code.
func IsRetryable(err error) bool {
	code := GetCode(err)
	if code == "" {
		return false
	}
	return retryableCodes[code]
}

// Convenience constructors – one per error category.

// ConfigError creates a CONFIG-category error.
func ConfigError(message string, cause error) *Error {
	return New(CodeConfig, message, cause)
}

// DetectError creates a DETECT-category error.
func DetectError(message string, cause error) *Error {
	return New(CodeDetect, message, cause)
}

// PatchError creates a PATCH-category error.
func PatchError(message string, cause error) *Error {
	return New(CodePatch, message, cause)
}

// GenerateError creates a GENERATE-category error.
func GenerateError(message string, cause error) *Error {
	return New(CodeGenerate, message, cause)
}

// ValidateError creates a VALIDATE-category error.
func ValidateError(message string, cause error) *Error {
	return New(CodeValidate, message, cause)
}

// LLMError creates an LLM-category error.
func LLMError(message string, cause error) *Error {
	return New(CodeLLM, message, cause)
}

// PublishError creates a PUBLISH-category error.
func PublishError(message string, cause error) *Error {
	return New(CodePublish, message, cause)
}

// SystemError creates a SYSTEM-category error.
func SystemError(message string, cause error) *Error {
	return New(CodeSystem, message, cause)
}
