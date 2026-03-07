package gin

import "testing"

func TestNewHandlerAnalyzer(t *testing.T) {
	a := NewHandlerAnalyzer()
	if a == nil {
		t.Error("expected non-nil analyzer")
	}
}
