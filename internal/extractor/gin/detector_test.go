package gin

import "testing"

func TestNewDetector(t *testing.T) {
	d := NewDetector()
	if d == nil {
		t.Error("expected non-nil detector")
	}
}
