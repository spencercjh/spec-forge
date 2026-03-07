package gin

import "testing"

func TestNewSchemaExtractor(t *testing.T) {
	e := NewSchemaExtractor()
	if e == nil {
		t.Error("expected non-nil extractor")
	}
}
