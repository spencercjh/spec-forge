package gin

// ASTParser parses Go AST to extract Gin routes.
type ASTParser struct {
	projectPath string
}

// NewASTParser creates a new ASTParser instance.
func NewASTParser(projectPath string) *ASTParser {
	return &ASTParser{projectPath: projectPath}
}
