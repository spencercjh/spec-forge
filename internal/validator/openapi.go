package validator

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

// SpecLoader defines the interface for loading and validating OpenAPI specs.
type SpecLoader interface {
	LoadAndValidate(ctx context.Context, specPath string) (*extractor.ValidateResult, error)
}

// openapiLoader implements SpecLoader using kin-openapi.
type openapiLoader struct{}

// LoadAndValidate loads and validates an OpenAPI spec from a file.
func (l *openapiLoader) LoadAndValidate(ctx context.Context, specPath string) (*extractor.ValidateResult, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	doc, err := loader.LoadFromFile(specPath)
	if err != nil {
		// Check if it's a file not found error
		if strings.Contains(err.Error(), "no such file") || strings.Contains(err.Error(), "cannot find") {
			return nil, fmt.Errorf("spec file not found: %s", specPath)
		}
		// Return validation result for parse errors
		return &extractor.ValidateResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("failed to parse OpenAPI spec: %v", err)},
		}, nil
	}

	// Validate the document
	if err := doc.Validate(ctx); err != nil {
		return &extractor.ValidateResult{
			Valid:  false,
			Errors: []string{formatValidationError(err)},
		}, nil
	}

	return &extractor.ValidateResult{
		Valid: true,
	}, nil
}

// formatValidationError formats validation errors for better readability.
func formatValidationError(err error) string {
	if err == nil {
		return ""
	}

	// Check for openapi3.SchemaError
	if schemaErr, ok := errors.AsType[*openapi3.SchemaError](err); ok {
		return fmt.Sprintf("validation error at %s: %s", schemaErr.JSONPointer(), schemaErr.Reason)
	}

	// Check for multi-error
	if errs, ok := err.(interface{ Unwrap() []error }); ok {
		var messages []string
		for _, e := range errs.Unwrap() {
			messages = append(messages, formatValidationError(e))
		}
		return strings.Join(messages, "; ")
	}

	return err.Error()
}
