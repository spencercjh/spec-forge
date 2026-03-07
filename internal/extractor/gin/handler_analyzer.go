package gin

import (
	"fmt"
	"go/ast"
	"go/token"
)

// HandlerAnalyzer analyzes Gin handler functions.
type HandlerAnalyzer struct {
	fset      *token.FileSet
	files     map[string]*ast.File
	typeCache map[string]*ast.TypeSpec
}

// NewHandlerAnalyzer creates a new HandlerAnalyzer instance.
func NewHandlerAnalyzer(fset *token.FileSet, files map[string]*ast.File) *HandlerAnalyzer {
	return &HandlerAnalyzer{
		fset:      fset,
		files:     files,
		typeCache: make(map[string]*ast.TypeSpec),
	}
}

// AnalyzeHandler analyzes a handler function and extracts information.
func (a *HandlerAnalyzer) AnalyzeHandler(fn *ast.FuncDecl) (*HandlerInfo, error) {
	info := &HandlerInfo{
		PathParams:   []ParamInfo{},
		QueryParams:  []ParamInfo{},
		HeaderParams: []ParamInfo{},
		Responses:    []ResponseInfo{},
	}

	// Inspect the function body
	if fn.Body == nil {
		return info, nil
	}

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			a.parseHandlerCall(node, info)
		}
		return true
	})

	return info, nil
}

// parseHandlerCall parses a call expression in a handler.
func (a *HandlerAnalyzer) parseHandlerCall(call *ast.CallExpr, info *HandlerInfo) {
	// Check for c.Param, c.Query, c.GetHeader, c.ShouldBindJSON, c.JSON
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	// Check if receiver is a context variable
	_, ok = sel.X.(*ast.Ident)
	if !ok {
		return
	}

	method := sel.Sel.Name

	switch method {
	case "Param":
		if len(call.Args) >= 1 {
			if name := extractStringLiteral(call.Args[0]); name != "" {
				info.PathParams = append(info.PathParams, ParamInfo{
					Name:     name,
					GoType:   "string",
					Required: true,
				})
			}
		}
	case "Query":
		if len(call.Args) >= 1 {
			if name := extractStringLiteral(call.Args[0]); name != "" {
				info.QueryParams = append(info.QueryParams, ParamInfo{
					Name:     name,
					GoType:   "string",
					Required: false,
				})
			}
		}
	case "DefaultQuery":
		if len(call.Args) >= 1 {
			if name := extractStringLiteral(call.Args[0]); name != "" {
				info.QueryParams = append(info.QueryParams, ParamInfo{
					Name:     name,
					GoType:   "string",
					Required: false,
				})
			}
		}
	case "GetHeader":
		if len(call.Args) >= 1 {
			if name := extractStringLiteral(call.Args[0]); name != "" {
				info.HeaderParams = append(info.HeaderParams, ParamInfo{
					Name:     name,
					GoType:   "string",
					Required: false,
				})
			}
		}
	case "ShouldBindJSON", "BindJSON", "ShouldBind":
		if len(call.Args) >= 1 {
			if typeName := extractTypeFromArg(call.Args[0]); typeName != "" {
				info.BodyType = typeName
			}
		}
	case "JSON":
		if len(call.Args) >= 2 {
			statusCode := extractStatusCode(call.Args[0])
			goType := extractTypeFromResponse(call.Args[1])
			info.Responses = append(info.Responses, ResponseInfo{
				StatusCode: statusCode,
				GoType:     goType,
			})
		}
	}
}

// extractTypeFromArg extracts type name from a binding argument.
func extractTypeFromArg(expr ast.Expr) string {
	// Pattern: &variable or &Struct{}
	unary, ok := expr.(*ast.UnaryExpr)
	if !ok || unary.Op != token.AND {
		return ""
	}

	// Check for composite literal: &Type{}
	if comp, ok := unary.X.(*ast.CompositeLit); ok {
		if ident, ok := comp.Type.(*ast.Ident); ok {
			return ident.Name
		}
		if sel, ok := comp.Type.(*ast.SelectorExpr); ok {
			if x, ok := sel.X.(*ast.Ident); ok {
				return x.Name + "." + sel.Sel.Name
			}
		}
	}

	// Check for variable: &variable
	if ident, ok := unary.X.(*ast.Ident); ok {
		return ident.Name // Return variable name, type would need type checking
	}

	return ""
}

// extractStatusCode extracts HTTP status code from expression.
func extractStatusCode(expr ast.Expr) int {
	// Integer literal
	if lit, ok := expr.(*ast.BasicLit); ok {
		if lit.Kind == token.INT {
			var code int
			// Try to parse
			if _, err := fmt.Sscanf(lit.Value, "%d", &code); err == nil {
				return code
			}
		}
	}

	// http.StatusOK reference
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		if x, ok := sel.X.(*ast.Ident); ok && x.Name == "http" {
			return statusCodeFromName(sel.Sel.Name)
		}
	}

	return 200 // Default
}

// statusCodeFromName converts http.StatusXxx to code.
func statusCodeFromName(name string) int {
	switch name {
	case "StatusOK":
		return 200
	case "StatusCreated":
		return 201
	case "StatusAccepted":
		return 202
	case "StatusNoContent":
		return 204
	case "StatusBadRequest":
		return 400
	case "StatusUnauthorized":
		return 401
	case "StatusForbidden":
		return 403
	case "StatusNotFound":
		return 404
	case "StatusInternalServerError":
		return 500
	case "StatusBadGateway":
		return 502
	case "StatusServiceUnavailable":
		return 503
	}
	return 200
}

// extractTypeFromResponse extracts type from response argument.
func extractTypeFromResponse(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.CompositeLit:
		if ident, ok := e.Type.(*ast.Ident); ok {
			return ident.Name
		}
		if sel, ok := e.Type.(*ast.SelectorExpr); ok {
			if x, ok := sel.X.(*ast.Ident); ok {
				return x.Name + "." + sel.Sel.Name
			}
		}
	case *ast.CallExpr:
		// gin.H or similar
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
			if sel.Sel.Name == "H" {
				return "map[string]any"
			}
		}
	}
	return ""
}
