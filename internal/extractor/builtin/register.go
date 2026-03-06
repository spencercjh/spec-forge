package builtin

import (
	"github.com/spencercjh/spec-forge/internal/extractor/gozero"
	"github.com/spencercjh/spec-forge/internal/extractor/spring"
)

func init() {
	// Register all built-in extractors
	Register(spring.ExtractorName, &spring.Extractor{})
	Register(gozero.ExtractorName, &gozero.Extractor{})
}
