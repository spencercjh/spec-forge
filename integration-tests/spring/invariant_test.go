//go:build e2e

package spring

import (
	"testing"

	"github.com/spencercjh/spec-forge/integration-tests/helpers"
)

// TestCriticalInvariants tests critical semantic invariants for Spring Boot generated specs
func TestCriticalInvariants(t *testing.T) {
	projectPath := "../maven-springboot-openapi-demo"

	skipIfProjectMissing(t, projectPath)

	outputDir := t.TempDir()
	specFile := generateSpec(t, projectPath, outputDir)
	validator := helpers.NewSpecValidator(t, specFile)

	t.Run("User Schema Must Have ID Field", func(t *testing.T) {
		validator.ValidateSchemaProperty("User", helpers.SchemaPropertyExpectation{
			Name: "id",
			Type: "integer",
		})
	})

	t.Run("User Schema Must Have Username Field", func(t *testing.T) {
		validator.ValidateSchemaProperty("User", helpers.SchemaPropertyExpectation{
			Name: "username",
			Type: "string",
		})
	})

	t.Run("PageResultUser Content Must Be User Array", func(t *testing.T) {
		validator.ValidateSchemaProperty("PageResultUser", helpers.SchemaPropertyExpectation{
			Name:     "content",
			Type:     "array",
			ItemType: "User",
		})
	})

	t.Run("Path Param ID Must Be Required", func(t *testing.T) {
		validator.ValidateParameterDetails("/api/v1/users/{id}", "get", []helpers.ParameterExpectation{
			{Name: "id", In: "path", Required: true},
		})
	})

	t.Run("GET Users Must Have Pagination Parameters", func(t *testing.T) {
		validator.ValidateParameterDetails("/api/v1/users", "get", []helpers.ParameterExpectation{
			{Name: "page", In: "query", Required: false},
			{Name: "size", In: "query", Required: false},
		})
	})

	t.Run("POST Users Request Must Reference User Schema", func(t *testing.T) {
		validator.ValidateRequestBodySchema("/api/v1/users", "post", "User")
	})

	t.Run("ApiResponseUser Must Reference User In Data Field", func(t *testing.T) {
		validator.ValidateSchemaProperty("ApiResponseUser", helpers.SchemaPropertyExpectation{
			Name: "data",
			Ref:  "User",
		})
	})

	t.Run("FileUploadResult Must Have Filename And Size Fields", func(t *testing.T) {
		validator.ValidateSchemaProperty("FileUploadResult", helpers.SchemaPropertyExpectation{
			Name: "filename",
			Type: "string",
		})
		validator.ValidateSchemaProperty("FileUploadResult", helpers.SchemaPropertyExpectation{
			Name: "size",
			Type: "integer",
		})
	})

	t.Run("All Expected Paths Must Exist", func(t *testing.T) {
		validator.ValidatePaths([]string{
			"/api/v1/users",
			"/api/v1/users/{id}",
			"/api/v1/users/{id}/profile",
			"/api/v1/users/upload",
		})
	})

	t.Run("All Expected Operations Must Have OperationId", func(t *testing.T) {
		validator.ValidateOperationFields("/api/v1/users", "get", true, false)
		validator.ValidateOperationFields("/api/v1/users", "post", true, false)
		validator.ValidateOperationFields("/api/v1/users/{id}", "get", true, false)
		validator.ValidateOperationFields("/api/v1/users/{id}/profile", "post", true, false)
		validator.ValidateOperationFields("/api/v1/users/upload", "post", true, false)
	})
}
