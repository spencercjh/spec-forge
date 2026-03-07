package gin

import "testing"

func TestNewPatcher(t *testing.T) {
	p := NewPatcher()
	if p == nil {
		t.Error("expected non-nil patcher")
	}
}
