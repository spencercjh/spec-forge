package gin

import (
	"go/ast"
	"go/token"
	"log/slog"
	"strings"
)

// inferParamTypes scans the handler body for strconv conversions and conditional
// comparisons that reveal the true type of query/path/header parameters previously
// extracted as string.
//
// Detected patterns:
//   - strconv.Atoi(c.Query("offset"))  → integer
//   - strconv.ParseInt(c.Query("id"), ...)  → integer
//   - strconv.ParseBool(c.Query("verbose"))  → boolean
//   - strconv.ParseFloat(c.Query("rate"), ...)  → number
//   - c.Query("x") == "true"  → boolean
func (a *HandlerAnalyzer) inferParamTypes(body *ast.BlockStmt, info *HandlerInfo) {
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			// Also check for conditional bool inference: c.Query("x") == "true"
			a.inferBoolFromComparison(n, info)
			return true
		}

		// Check for strconv.Xxx(c.Query/Param(...))
		a.inferTypeFromStrconv(call, info)
		return true
	})
}

// inferTypeFromStrconv checks if a call is strconv.Atoi/ParseInt/ParseBool/ParseFloat
// wrapping a c.Query/c.Param/c.DefaultQuery call, and updates the parameter's GoType.
func (a *HandlerAnalyzer) inferTypeFromStrconv(call *ast.CallExpr, info *HandlerInfo) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	x, ok := sel.X.(*ast.Ident)
	if !ok || x.Name != "strconv" {
		return
	}

	// Determine target type from function name
	targetType := ""
	switch sel.Sel.Name {
	case "Atoi", "ParseInt":
		targetType = goTypeInteger
	case "ParseBool":
		targetType = goTypeBoolean
	case "ParseFloat":
		targetType = goTypeNumber
	default:
		return
	}

	// Find c.Query/c.Param/c.DefaultQuery in the arguments
	if len(call.Args) < 1 {
		return
	}

	paramName := a.findContextQueryParam(call.Args[0])
	if paramName != "" {
		a.updateParamType(info, paramName, targetType)
	}
}

// inferBoolFromComparison checks for patterns like c.Query("x") == "true"
// or verbose == "true" (where verbose was assigned from c.Query)
// which suggest the parameter is boolean-typed.
func (a *HandlerAnalyzer) inferBoolFromComparison(n ast.Node, info *HandlerInfo) {
	binExpr, ok := n.(*ast.BinaryExpr)
	if !ok || binExpr.Op.String() != "==" {
		return
	}

	// Check if one side is "true"/"false" literal
	var exprSide ast.Expr
	var literalSide ast.Expr

	litX, okX := binExpr.X.(*ast.BasicLit)
	litY, okY := binExpr.Y.(*ast.BasicLit)
	if okX && litX.Kind == token.STRING {
		literalSide = binExpr.X
		exprSide = binExpr.Y
	} else if okY && litY.Kind == token.STRING {
		literalSide = binExpr.Y
		exprSide = binExpr.X
	} else {
		return
	}

	lit, ok := literalSide.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return
	}
	value := strings.Trim(lit.Value, `"`)
	if value != "true" && value != "false" {
		return
	}

	// Check if the expression side is a c.Query/c.Param call
	paramName := a.findContextQueryParam(exprSide)
	if paramName != "" {
		a.updateParamType(info, paramName, goTypeBoolean)
		return
	}

	// Also check if the expression side is a variable assigned from c.Query/c.Param
	// We check the existing query params — if a variable name matches a query param name,
	// this comparison suggests the param is boolean
	if ident, ok := exprSide.(*ast.Ident); ok {
		for i := range info.QueryParams {
			if info.QueryParams[i].Name == ident.Name {
				info.QueryParams[i].GoType = goTypeBoolean
				slog.Debug("Inferred parameter type from comparison", "param", ident.Name, "type", goTypeBoolean)
				return
			}
		}
	}
}

// findContextQueryParam checks if an expression is c.Query("name") or c.Param("name")
// and returns the parameter name. Recursively unwraps simple wrappers (e.g., type casts).
func (a *HandlerAnalyzer) findContextQueryParam(expr ast.Expr) string {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return ""
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	if _, ok := sel.X.(*ast.Ident); !ok {
		return ""
	}

	switch sel.Sel.Name {
	case "Query", "DefaultQuery", "Param": //nolint:goconst // AST method name literals
		if len(call.Args) >= 1 {
			return extractStringLiteral(call.Args[0])
		}
	}
	return ""
}

// updateParamType finds an existing parameter by name and updates its GoType.
// Searches query, path, and header params. Does nothing if not found.
func (a *HandlerAnalyzer) updateParamType(info *HandlerInfo, paramName, targetType string) {
	for i := range info.QueryParams {
		if info.QueryParams[i].Name == paramName {
			info.QueryParams[i].GoType = targetType
			slog.Debug("Inferred parameter type", "param", paramName, "type", targetType)
			return
		}
	}
	for i := range info.PathParams {
		if info.PathParams[i].Name == paramName {
			info.PathParams[i].GoType = targetType
			slog.Debug("Inferred parameter type", "param", paramName, "type", targetType)
			return
		}
	}
	for i := range info.HeaderParams {
		if info.HeaderParams[i].Name == paramName {
			info.HeaderParams[i].GoType = targetType
			slog.Debug("Inferred parameter type", "param", paramName, "type", targetType)
			return
		}
	}
}

// isFormRelatedWord checks if text contains form-related words as whole words.
// This prevents false positives like "performAction", "platform", "information".
func isFormRelatedWord(text string) bool {
	lower := strings.ToLower(text)

	// Define whole-word patterns to match
	patterns := []string{"form", "formdata", "form-data", "multipart"}

	for _, pattern := range patterns {
		// Check for pattern preceded by start of string or non-letter
		// and followed by end of string or non-letter
		for i := 0; i <= len(lower)-len(pattern); i++ {
			if lower[i:i+len(pattern)] == pattern {
				// Check prefix (start of string or non-letter)
				prefixOK := i == 0 || !isLetter(lower[i-1])
				// Check suffix (end of string or non-letter)
				suffixOK := i+len(pattern) == len(lower) || !isLetter(lower[i+len(pattern)])
				if prefixOK && suffixOK {
					return true
				}
			}
		}
	}

	return false
}

// isLetter checks if a byte is a letter (a-z).
func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
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
	if varName[0] >= 'a' && varName[0] <= 'z' {
		return string(varName[0]-'a'+'A') + varName[1:]
	}

	return varName
}
