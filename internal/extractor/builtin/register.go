package builtin

import (
	"github.com/spencercjh/spec-forge/internal/extractor/gin"
	"github.com/spencercjh/spec-forge/internal/extractor/gozero"
	"github.com/spencercjh/spec-forge/internal/extractor/grpcprotoc"
	"github.com/spencercjh/spec-forge/internal/extractor/spring"
)

func init() {
	// Register all built-in extractors
	Register(spring.ExtractorName, spring.NewExtractor())
	Register(gozero.ExtractorName, gozero.NewExtractor())
	Register(grpcprotoc.ExtractorName, grpcprotoc.NewExtractor())
	Register(gin.ExtractorName, gin.NewExtractor())
}
