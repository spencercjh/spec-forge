package errors

import (
	"testing"
)

func TestRecoveryHint(t *testing.T) {
	knownCodes := []string{
		CodeConfig,
		CodeDetect,
		CodePatch,
		CodeGenerate,
		CodeValidate,
		CodeLLM,
		CodePublish,
		CodeSystem,
	}
	for _, code := range knownCodes {
		t.Run(code, func(t *testing.T) {
			hint := RecoveryHint(code)
			if hint == "" {
				t.Errorf("RecoveryHint(%q) returned empty string", code)
			}
		})
	}
}

func TestRecoveryHint_Unknown(t *testing.T) {
	hint := RecoveryHint("UNKNOWN")
	if hint != "" {
		t.Errorf("RecoveryHint(\"UNKNOWN\") = %q, want empty string", hint)
	}
}

func TestRetryableCodes(t *testing.T) {
	retryable := []string{CodeLLM, CodePublish, CodeSystem}
	for _, code := range retryable {
		if !retryableCodes[code] {
			t.Errorf("expected code %q to be retryable", code)
		}
	}

	notRetryable := []string{CodeConfig, CodeDetect, CodePatch, CodeGenerate, CodeValidate}
	for _, code := range notRetryable {
		if retryableCodes[code] {
			t.Errorf("expected code %q to NOT be retryable", code)
		}
	}
}
