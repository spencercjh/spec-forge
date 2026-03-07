package gin

import "testing"

func TestNewGinExtractor(t *testing.T) {
	e := NewGinExtractor()
	if e == nil {
		t.Error("expected non-nil extractor")
	}
}
