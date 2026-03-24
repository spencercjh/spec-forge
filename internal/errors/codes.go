package errors

// Error category codes used to classify spec-forge errors.
const (
	// CodeConfig indicates a configuration error (invalid config, missing env vars).
	CodeConfig = "CONFIG"

	// CodeDetect indicates a detection error (framework not detected, invalid project structure).
	CodeDetect = "DETECT"

	// CodePatch indicates a patching error (dependency injection failed, build file issues).
	CodePatch = "PATCH"

	// CodeGenerate indicates a generation error (spec generation failed).
	CodeGenerate = "GENERATE"

	// CodeValidate indicates a validation error (OpenAPI spec invalid or non-compliant).
	CodeValidate = "VALIDATE"

	// CodeLLM indicates an LLM/enrichment error (AI provider errors, rate limits).
	CodeLLM = "LLM"

	// CodePublish indicates a publishing error (upload to platform failed, credentials issues).
	CodePublish = "PUBLISH"

	// CodeSystem indicates a system-level error (file I/O, command execution, timeout).
	CodeSystem = "SYSTEM"
)

// recoveryHints maps error codes to user-facing recovery instructions.
var recoveryHints = map[string]string{
	CodeConfig:   "Check your .spec-forge.yaml configuration file and ensure all required environment variables are set",
	CodeDetect:   "Verify the project structure and ensure it contains the expected build files (pom.xml, build.gradle, go.mod, .proto files)",
	CodePatch:    "Check that build files are writable and the project has correct permissions",
	CodeGenerate: "Check build logs for compilation errors and ensure all dependencies are available",
	CodeValidate: "Review the generated OpenAPI spec for compliance issues and fix any schema errors",
	CodeLLM:      "Check your API key, model name, and network connectivity; consider retrying as this may be a transient error",
	CodePublish:  "Verify your publishing credentials and network connectivity; consider retrying",
	CodeSystem:   "Check system resources, file permissions, and ensure required tools are installed; consider retrying",
}

// retryableCodes lists error codes for which retry is generally appropriate.
var retryableCodes = map[string]bool{
	CodeLLM:     true,
	CodePublish: true,
	CodeSystem:  true,
}

// RecoveryHint returns a user-facing recovery hint for the given error code.
// Returns an empty string for unknown codes.
func RecoveryHint(code string) string {
	return recoveryHints[code]
}
