package gin

import "testing"

func TestNewASTParser(t *testing.T) {
	p := NewASTParser("/tmp")
	if p == nil {
		t.Error("expected non-nil parser")
	}
}
