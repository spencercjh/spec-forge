//go:build e2e

package helpers

// ValidationConfig configures comprehensive spec validation
type ValidationConfig struct {
	ExpectedPaths   []string
	Operations      []OperationConfig
	ExpectedSchemas []string
}

// OperationConfig configures operation validation
type OperationConfig struct {
	Path                    string
	Method                  string
	WantOperationID         bool
	WantSummary             bool
	ExpectedResponseCodes   []string
	ExpectedParams          []string
	ValidateResponseContent string
	WantRequestBody         string
}

// ParameterExpectation configures parameter validation
type ParameterExpectation struct {
	Name     string
	In       string
	Required bool
}

// SchemaPropertyExpectation configures schema property validation
type SchemaPropertyExpectation struct {
	Name     string
	Type     string
	ItemType string
	Ref      string
}

// ResponseSchemaExpectation configures response schema validation
type ResponseSchemaExpectation struct {
	Code        string
	ContentType string
	SchemaRef   string
}

// GoldenSnapshot represents a golden file snapshot
type GoldenSnapshot struct {
	Name     string
	Path     string // Path in OpenAPI spec (e.g., "paths./api/v1/users.get")
	File     string // Golden file path
	Optional bool   // If true, test won't fail if missing (for unstable features)
}
