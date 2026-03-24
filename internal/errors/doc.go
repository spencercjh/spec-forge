// Package errors provides a unified error classification system for spec-forge.
//
// All spec-forge errors are classified into one of eight categories:
//
//   - CONFIG: Configuration errors (invalid config, missing env vars)
//   - DETECT: Detection errors (framework not found, invalid project structure)
//   - PATCH: Patching errors (dependency injection failed, build file issues)
//   - GENERATE: Generation errors (spec generation failed)
//   - VALIDATE: Validation errors (OpenAPI spec invalid)
//   - LLM: LLM/enrichment errors (AI provider errors, rate limits)
//   - PUBLISH: Publishing errors (upload failed, credentials issues)
//   - SYSTEM: System errors (file I/O, command execution, timeout)
//
// # Usage
//
// Creating errors:
//
//	err := forgeerrors.DetectError("no pom.xml found", nil)
//	err := forgeerrors.SystemError("command failed", cause)
//	err := forgeerrors.New(forgeerrors.CodeGenerate, "maven failed", cause)
//
// Checking error categories:
//
//	if forgeerrors.IsCode(err, forgeerrors.CodeLLM) {
//	    // handle LLM error
//	}
//
//	code := forgeerrors.GetCode(err)
//
// Adding context:
//
//	err := forgeerrors.DetectError("no build file", nil).
//	    WithContext("path", projectPath).
//	    WithContext("searched", []string{"pom.xml", "build.gradle"})
//
// Recovery hints:
//
//	hint := forgeerrors.RecoveryHint(forgeerrors.CodeConfig)
//	// or from an error:
//	if fe, ok := err.(*forgeerrors.Error); ok {
//	    fmt.Println(fe.Hint())
//	}
package errors
