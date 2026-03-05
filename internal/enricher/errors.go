package enricher

import (
	"errors"
	"fmt"
)

// Error type constants
const (
	ErrorTypeConfig   = "config"
	ErrorTypeLLMCall  = "llm_call"
	ErrorTypeParse    = "parse"
	ErrorTypeTemplate = "template"
)

// EnrichmentError represents an error during the enrichment process
type EnrichmentError struct {
	Type    string
	Message string
	Cause   error
}

// Error implements the error interface
func (e *EnrichmentError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("enrichment error [%s]: %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("enrichment error [%s]: %s", e.Type, e.Message)
}

// Unwrap returns the underlying cause for errors.Is/As
func (e *EnrichmentError) Unwrap() error {
	return e.Cause
}

// NewEnrichmentError creates a new EnrichmentError
func NewEnrichmentError(errorType, message string, cause error) *EnrichmentError {
	return &EnrichmentError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
	}
}

// IsConfigError checks if the error is a configuration error
func IsConfigError(err error) bool {
	var enrichErr *EnrichmentError
	if errors.As(err, &enrichErr) {
		return enrichErr.Type == ErrorTypeConfig
	}
	return false
}
