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

	// Build variable type map for type inference
	varTypeMap := a.buildVarTypeMap(fn.Body)

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if node, ok := n.(*ast.CallExpr); ok {
			a.parseHandlerCall(node, info, varTypeMap)
		}
		return true
	})

	return info, nil
}

// buildVarTypeMap builds a map of variable names to their types.
func (a *HandlerAnalyzer) buildVarTypeMap(body *ast.BlockStmt) map[string]string {
	varTypeMap := make(map[string]string)

	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.DeclStmt:
			// Extract type from var declarations like: var req CreateUserRequest
			if genDecl, ok := node.Decl.(*ast.GenDecl); ok && genDecl.Tok == token.VAR {
				for _, spec := range genDecl.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						for i, name := range valueSpec.Names {
							if i < len(valueSpec.Values) {
								// Check if value is a composite literal with type
								if comp, ok := valueSpec.Values[i].(*ast.CompositeLit); ok {
									if ident, ok := comp.Type.(*ast.Ident); ok {
										varTypeMap[name.Name] = ident.Name
									}
								}
							}
							// Also check explicit type
							if valueSpec.Type != nil {
								if ident, ok := valueSpec.Type.(*ast.Ident); ok {
									varTypeMap[name.Name] = ident.Name
								}
							}
						}
					}
				}
			}
		case *ast.AssignStmt:
			// Extract type from short declarations like: req := CreateUserRequest{}
			if node.Tok == token.DEFINE {
				for i, lhs := range node.Lhs {
					if i < len(node.Rhs) {
						if ident, ok := lhs.(*ast.Ident); ok {
							// Check if RHS is a composite literal with type
							if comp, ok := node.Rhs[i].(*ast.CompositeLit); ok {
								if typeIdent, ok := comp.Type.(*ast.Ident); ok {
									varTypeMap[ident.Name] = typeIdent.Name
								}
							} else {
								// Try to infer type from variable name using heuristic
								// e.g., "user" -> "User", "result" -> "Result"
								if inferredType := inferTypeFromVarName(ident.Name); inferredType != "" {
									varTypeMap[ident.Name] = inferredType
								}
							}
						}
					}
				}
			}
		}
		return true
	})

	return varTypeMap
}

// inferTypeFromVarName tries to infer type from variable name using common patterns.
// e.g., "user" -> "User", "users" -> "User", "result" -> "Result"
func inferTypeFromVarName(varName string) string {
	if varName == "" {
		return ""
	}

	// Common variable name -> type mappings
	mappings := map[string]string{
		"user":   "User",
		"users":  "User",
		"req":    "",
		"result": "",
		"data":   "",
		"item":   "",
		"items":  "",
	}

	if typ, ok := mappings[varName]; ok {
		return typ
	}

	// Try to capitalize first letter (heuristic)
	// e.g., "pageResult" -> "PageResult"
	if varName != "" && varName[0] >= 'a' && varName[0] <= 'z' {
		return string(varName[0]-'a'+'A') + varName[1:]
	}

	return varName
}

// parseHandlerCall parses a call expression in a handler.
func (a *HandlerAnalyzer) parseHandlerCall(call *ast.CallExpr, info *HandlerInfo, varTypeMap map[string]string) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	// Check if receiver is a context variable
	if _, ok = sel.X.(*ast.Ident); !ok {
		return
	}

	method := sel.Sel.Name

	// Categorize methods and dispatch to specialized handlers
	switch method {
	case "Param":
		a.extractParam(call, info)
	case "Query", "DefaultQuery":
		a.extractQueryParam(call, info, method == "Query")
	case "GetHeader":
		a.extractHeaderParam(call, info)
	case "ShouldBindJSON", "BindJSON", "ShouldBind", "Bind":
		a.extractBodyType(call, info, varTypeMap)
	case "ShouldBindXML", "BindXML":
		a.extractBodyType(call, info, varTypeMap)
	case "ShouldBindYAML", "BindYAML":
		a.extractBodyType(call, info, varTypeMap)
	case "ShouldBindQuery", "BindQuery":
		a.extractBodyType(call, info, varTypeMap)
	case "PostForm":
		a.extractFormParam(call, info, true)
	case "DefaultPostForm":
		a.extractFormParam(call, info, false)
	case "FormFile":
		a.extractFileParam(call, info)
	case "JSON", "XML", "YAML":
		a.extractResponse(call, info, varTypeMap, "")
	case "String":
		a.extractResponse(call, info, varTypeMap, "string")
	case "Data":
		a.extractResponse(call, info, varTypeMap, "binary")
	case "Redirect":
		a.extractResponse(call, info, varTypeMap, "")
	}
}

// extractParam extracts path parameter from c.Param() call.
func (a *HandlerAnalyzer) extractParam(call *ast.CallExpr, info *HandlerInfo) {
	if len(call.Args) < 1 {
		return
	}
	if name := extractStringLiteral(call.Args[0]); name != "" {
		info.PathParams = append(info.PathParams, ParamInfo{
			Name:     name,
			GoType:   "string",
			Required: true,
		})
	}
}

// extractQueryParam extracts query parameter from c.Query() or c.DefaultQuery() call.
func (a *HandlerAnalyzer) extractQueryParam(call *ast.CallExpr, info *HandlerInfo, required bool) {
	if len(call.Args) < 1 {
		return
	}
	if name := extractStringLiteral(call.Args[0]); name != "" {
		info.QueryParams = append(info.QueryParams, ParamInfo{
			Name:     name,
			GoType:   "string",
			Required: required,
		})
	}
}

// extractHeaderParam extracts header parameter from c.GetHeader() call.
func (a *HandlerAnalyzer) extractHeaderParam(call *ast.CallExpr, info *HandlerInfo) {
	if len(call.Args) < 1 {
		return
	}
	if name := extractStringLiteral(call.Args[0]); name != "" {
		info.HeaderParams = append(info.HeaderParams, ParamInfo{
			Name:     name,
			GoType:   "string",
			Required: false,
		})
	}
}

// extractBodyType extracts body type from binding calls like c.ShouldBindJSON().
func (a *HandlerAnalyzer) extractBodyType(call *ast.CallExpr, info *HandlerInfo, varTypeMap map[string]string) {
	if len(call.Args) < 1 {
		return
	}
	if typeName := extractTypeFromArg(call.Args[0], varTypeMap); typeName != "" {
		info.BodyType = typeName
	}
}

// extractFormParam extracts form parameter from c.PostForm() or c.DefaultPostForm() call.
func (a *HandlerAnalyzer) extractFormParam(call *ast.CallExpr, info *HandlerInfo, required bool) {
	if len(call.Args) < 1 {
		return
	}
	if name := extractStringLiteral(call.Args[0]); name != "" {
		info.QueryParams = append(info.QueryParams, ParamInfo{
			Name:     name,
			GoType:   "string",
			Required: required,
		})
	}
}

// extractFileParam extracts file parameter from c.FormFile() call.
func (a *HandlerAnalyzer) extractFileParam(call *ast.CallExpr, info *HandlerInfo) {
	if len(call.Args) < 1 {
		return
	}
	if name := extractStringLiteral(call.Args[0]); name != "" {
		info.QueryParams = append(info.QueryParams, ParamInfo{
			Name:     name,
			GoType:   "file",
			Required: true,
		})
	}
}

// extractResponse extracts response information from c.JSON(), c.XML(), etc. calls.
func (a *HandlerAnalyzer) extractResponse(call *ast.CallExpr, info *HandlerInfo, varTypeMap map[string]string, goType string) {
	if len(call.Args) < 2 {
		return
	}
	statusCode := extractStatusCode(call.Args[0])
	if goType == "" {
		goType = extractTypeFromResponse(call.Args[1], varTypeMap)
	}
	info.Responses = append(info.Responses, ResponseInfo{
		StatusCode: statusCode,
		GoType:     goType,
	})
}

// extractTypeFromArg extracts type name from a binding argument.
func extractTypeFromArg(expr ast.Expr, varTypeMap map[string]string) string {
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
		// First check if we have type info from variable declaration
		if varType, exists := varTypeMap[ident.Name]; exists {
			return varType
		}
		// Fall back to variable name
		return ident.Name
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
func extractTypeFromResponse(expr ast.Expr, varTypeMap map[string]string) string {
	switch e := expr.(type) {
	case *ast.Ident:
		// If it's a variable, try to get its actual type from the map
		if actualType, ok := varTypeMap[e.Name]; ok {
			return actualType
		}
		return e.Name
	case *ast.CompositeLit:
		// First check if this is ApiResponse{Data: user} pattern
		// Try to extract the type from the Data field
		for _, elt := range e.Elts {
			if kv, ok := elt.(*ast.KeyValueExpr); ok {
				if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Data" {
					// Found Data field, recursively extract its type
					dataType := extractTypeFromResponse(kv.Value, varTypeMap)
					if dataType != "" {
						return dataType
					}
				}
			}
		}
		// Fall back to extracting the composite literal type itself
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
				return ginHType
			}
		}
		// Could be a function call that returns a type, try to extract from return type
		if ident, ok := e.Fun.(*ast.Ident); ok {
			if actualType, ok := varTypeMap[ident.Name]; ok {
				return actualType
			}
		}
	case *ast.UnaryExpr:
		// &variable -> dereference and get the type
		if e.Op == token.AND {
			return extractTypeFromResponse(e.X, varTypeMap)
		}
	}
	return ""
}
