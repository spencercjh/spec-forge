package enricher

import (
	"errors"
	"fmt"

	forgeerrors "github.com/spencercjh/spec-forge/internal/errors"
)

// Error type constants
const (
	ErrorTypeConfig   = "config"
	ErrorTypeLLMCall  = "llm_call"
	ErrorTypeParse    = "parse"
	ErrorTypeTemplate = "template"
)

// enricherTypeToCode maps enricher error type constants to unified category codes.
var enricherTypeToCode = map[string]string{
	ErrorTypeConfig:   forgeerrors.CodeConfig,
	ErrorTypeLLMCall:  forgeerrors.CodeLLM,
	ErrorTypeParse:    forgeerrors.CodeLLM,
	ErrorTypeTemplate: forgeerrors.CodeLLM,
}

// EnrichmentError represents an error during the enrichment process
type EnrichmentError struct {
	Type       string
	Message    string
	Cause      error
	classified *forgeerrors.Error
}

// Error implements the error interface
func (e *EnrichmentError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("enrichment error [%s]: %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("enrichment error [%s]: %s", e.Type, e.Message)
}

// Unwrap returns the underlying cause for errors.Is/As. When a classified error
// is present, it is returned so that forgeerrors.IsCode works through the chain.
func (e *EnrichmentError) Unwrap() error {
	if e.classified != nil {
		return e.classified
	}
	return e.Cause
}

// NewEnrichmentError creates a new EnrichmentError
func NewEnrichmentError(errorType, message string, cause error) *EnrichmentError {
	code, ok := enricherTypeToCode[errorType]
	if !ok {
		code = forgeerrors.CodeLLM
	}
	return &EnrichmentError{
		Type:       errorType,
		Message:    message,
		Cause:      cause,
		classified: forgeerrors.New(code, fmt.Sprintf("enrichment error [%s]: %s", errorType, message), cause),
	}
}

// IsConfigError checks if the error is a configuration error
func IsConfigError(err error) bool {
	if enrichErr, ok := errors.AsType[*EnrichmentError](err); ok {
		return enrichErr.Type == ErrorTypeConfig
	}
	return false
}
