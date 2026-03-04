// Package validator provides OpenAPI spec validation functionality.
package validator

import (
	"context"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

// Validator validates OpenAPI specifications.
type Validator struct {
	loader SpecLoader
}

// NewValidator creates a new Validator instance.
func NewValidator() *Validator {
	return &Validator{
		loader: &openapiLoader{},
	}
}

// Validate validates an OpenAPI spec file.
func (v *Validator) Validate(ctx context.Context, specPath string) (*extractor.ValidateResult, error) {
	return v.loader.LoadAndValidate(ctx, specPath)
}
