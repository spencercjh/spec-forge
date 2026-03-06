package publisher

import (
	"errors"
	"testing"
)

func TestNewPublisher(t *testing.T) {
	tests := []struct {
		name        string
		destType    string
		wantName    string
		wantErr     bool
		wantErrType error
	}{
		{
			name:     "local publisher lowercase",
			destType: "local",
			wantName: "local",
			wantErr:  false,
		},
		{
			name:     "local publisher uppercase",
			destType: "LOCAL",
			wantName: "local",
			wantErr:  false,
		},
		{
			name:     "local publisher mixed case",
			destType: "Local",
			wantName: "local",
			wantErr:  false,
		},
		{
			name:     "local publisher with whitespace",
			destType: "  local  ",
			wantName: "local",
			wantErr:  false,
		},
		{
			name:     "empty string defaults to local",
			destType: "",
			wantName: "local",
			wantErr:  false,
		},
		{
			name:     "readme publisher lowercase",
			destType: "readme",
			wantName: "readme",
			wantErr:  false,
		},
		{
			name:     "readme publisher uppercase",
			destType: "README",
			wantName: "readme",
			wantErr:  false,
		},
		{
			name:     "readme publisher mixed case",
			destType: "ReadMe",
			wantName: "readme",
			wantErr:  false,
		},
		{
			name:        "unknown publisher type",
			destType:    "unknown",
			wantErr:     true,
			wantErrType: ErrUnknownPublisher,
		},
		{
			name:        "postman publisher not implemented",
			destType:    "postman",
			wantErr:     true,
			wantErrType: ErrUnknownPublisher,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pub, err := NewPublisher(tt.destType)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewPublisher(%q) expected error, got nil", tt.destType)
					return
				}
				if tt.wantErrType != nil && !errors.Is(err, tt.wantErrType) {
					t.Errorf("NewPublisher(%q) error = %v, want error wrapping %v", tt.destType, err, tt.wantErrType)
				}
				return
			}

			if err != nil {
				t.Errorf("NewPublisher(%q) unexpected error: %v", tt.destType, err)
				return
			}

			if pub.Name() != tt.wantName {
				t.Errorf("NewPublisher(%q).Name() = %q, want %q", tt.destType, pub.Name(), tt.wantName)
			}
		})
	}
}

func TestErrUnknownPublisher(t *testing.T) {
	// Test that ErrUnknownPublisher is correctly defined and can be used with errors.Is
	err := errors.New("unknown publisher type: \"test\"")
	if errors.Is(err, ErrUnknownPublisher) {
		// This won't match because err doesn't wrap ErrUnknownPublisher
		t.Error("plain error should not match ErrUnknownPublisher")
	}

	// Test that NewPublisher returns an error that wraps ErrUnknownPublisher
	_, err = NewPublisher("invalid")
	if !errors.Is(err, ErrUnknownPublisher) {
		t.Errorf("NewPublisher(\"invalid\") error should wrap ErrUnknownPublisher, got: %v", err)
	}
}
