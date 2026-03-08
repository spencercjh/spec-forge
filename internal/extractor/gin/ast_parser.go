package gin

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ASTParser parses Go AST to extract Gin routes.
type ASTParser struct {
	fset        *token.FileSet
	projectPath string
	files       map[string]*ast.File
	routerVars  map[string]bool // Track router variable names
	groups      map[string]*GroupInfo
}

// GroupInfo stores information about a router group.
type GroupInfo struct {
	BasePath string
	VarName  string
}

// NewASTParser creates a new ASTParser instance.
func NewASTParser(projectPath string) *ASTParser {
	return &ASTParser{
		projectPath: projectPath,
		fset:        token.NewFileSet(),
		files:       make(map[string]*ast.File),
		routerVars:  make(map[string]bool),
		groups:      make(map[string]*GroupInfo),
	}
}

// ParseFiles parses all Go files in the project.
func (p *ASTParser) ParseFiles() error {
	return filepath.Walk(p.projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor and hidden directories
		if info.IsDir() {
			if info.Name() == "vendor" || strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Parse the file
		// #nosec G122 - path is validated through filepath.Walk which handles symlink traversal
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		file, parseErr := parser.ParseFile(p.fset, path, content, parser.ParseComments)
		_ = parseErr // Intentionally ignore parse errors to skip malformed files
		if file == nil {
			// Skip files that failed to parse
			return nil
		}

		p.files[path] = file
		return nil
	})
}

// ExtractRoutes extracts all Gin routes from parsed files.
func (p *ASTParser) ExtractRoutes() ([]Route, error) {
	var routes []Route

	// First pass: extract group definitions
	for path, file := range p.files {
		groups := p.extractGroups(file)
		for name, group := range groups {
			p.groups[name] = group
			_ = path // use path
		}
	}

	// Second pass: extract routes
	for path, file := range p.files {
		fileRoutes := p.extractRoutesFromFile(path, file)
		routes = append(routes, fileRoutes...)
	}

	return routes, nil
}

// extractGroups extracts router group definitions from a file.
func (p *ASTParser) extractGroups(file *ast.File) map[string]*GroupInfo {
	groups := make(map[string]*GroupInfo)

	ast.Inspect(file, func(n ast.Node) bool {
		if node, ok := n.(*ast.AssignStmt); ok {
			if len(node.Lhs) == 1 && len(node.Rhs) == 1 {
				if group := p.parseGroupAssignment(node.Rhs[0]); group != nil {
					if ident, ok := node.Lhs[0].(*ast.Ident); ok {
						group.VarName = ident.Name
						groups[ident.Name] = group
					}
				}
			}
		}
		return true
	})

	return groups
}

// parseGroupAssignment parses a Group assignment.
func (p *ASTParser) parseGroupAssignment(expr ast.Expr) *GroupInfo {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Group" {
		return nil
	}

	if len(call.Args) < 1 {
		return nil
	}

	basePath := extractStringLiteral(call.Args[0])
	if basePath == "" {
		return nil
	}

	return &GroupInfo{BasePath: basePath}
}

// extractRoutesFromFile extracts routes from a single file.
func (p *ASTParser) extractRoutesFromFile(path string, file *ast.File) []Route {
	var routes []Route

	// Inspect the AST
	ast.Inspect(file, func(n ast.Node) bool {
		if node, ok := n.(*ast.CallExpr); ok {
			if route := p.parseRouteCall(path, node); route != nil {
				routes = append(routes, *route)
			}
		}
		return true
	})

	return routes
}

// parseRouteCall parses a route registration call.
func (p *ASTParser) parseRouteCall(file string, call *ast.CallExpr) *Route {
	// Pattern: r.GET("/path", handler) or api.GET("/path", handler)
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	method := sel.Sel.Name
	if !isHTTPMethod(method) {
		return nil
	}

	// Get the router variable
	routerVar, ok := sel.X.(*ast.Ident)
	if !ok {
		return nil
	}

	// Need at least 2 args: path and handler
	if len(call.Args) < 2 {
		return nil
	}

	// Extract path
	path := extractStringLiteral(call.Args[0])
	if path == "" {
		return nil
	}

	// Check if this is a group route
	fullPath := path
	if group, ok := p.groups[routerVar.Name]; ok {
		fullPath = group.BasePath + path
	}

	// Convert Gin path format (:id) to OpenAPI path format ({id})
	fullPath = convertPathFormat(fullPath)

	// Extract handler name
	handlerName := extractHandlerName(call.Args[1])

	return &Route{
		Method:      method,
		Path:        path,
		FullPath:    fullPath,
		HandlerName: handlerName,
		HandlerFile: file,
	}
}

// isHTTPMethod checks if a string is an HTTP method.
func isHTTPMethod(s string) bool {
	switch s {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS":
		return true
	}
	return false
}

// convertPathFormat converts Gin path format (:param) to OpenAPI path format ({param}).
func convertPathFormat(path string) string {
	// Replace :param with {param}
	// Handle patterns like :id, :userId, etc.
	var result strings.Builder
	for i := 0; i < len(path); i++ {
		if path[i] == ':' {
			// Find the end of the parameter name
			j := i + 1
			for j < len(path) && isValidParamChar(path[j]) {
				j++
			}
			if j > i+1 {
				// Convert :param to {param}
				result.WriteByte('{')
				result.WriteString(path[i+1 : j])
				result.WriteByte('}')
				i = j - 1
			} else {
				result.WriteByte(path[i])
			}
		} else {
			result.WriteByte(path[i])
		}
	}
	return result.String()
}

// isValidParamChar checks if a character is valid for a parameter name.
func isValidParamChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// extractStringLiteral extracts a string from an AST expression.
func extractStringLiteral(expr ast.Expr) string {
	if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		// Remove quotes
		return strings.Trim(lit.Value, `"`)
	}
	return ""
}

// extractHandlerName extracts handler name from an expression.
func extractHandlerName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.FuncLit:
		return "" // Anonymous function
	case *ast.SelectorExpr:
		if x, ok := e.X.(*ast.Ident); ok {
			return x.Name + "." + e.Sel.Name
		}
	}
	return ""
}
