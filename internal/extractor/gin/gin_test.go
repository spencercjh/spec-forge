package gin

import "testing"

func TestNewExtractor(t *testing.T) {
	e := NewExtractor()
	if e == nil {
		t.Error("expected non-nil extractor")
	}
	if e.Name() != ExtractorName {
		t.Errorf("expected name %q, got %q", ExtractorName, e.Name())
	}
}
