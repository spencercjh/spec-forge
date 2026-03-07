package gin

import "testing"

func TestNewGenerator(t *testing.T) {
	g := NewGenerator()
	if g == nil {
		t.Error("expected non-nil generator")
	}
}
