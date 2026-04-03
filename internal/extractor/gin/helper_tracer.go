package gin

import (
	"go/ast"
	"log/slog"
	"slices"
)

// knownResponseHelperNames lists common response helper function names.
// These are standalone functions (not methods on gin.Context) that handle
// responses, typically taking *gin.Context as the first parameter.
var knownResponseHelperNames = []string{
	"done", "respond", "response",
	"writeJSON", "sendJSON", "reply",
}

// isKnownResponseHelper checks if a function name is a known response helper.
func isKnownResponseHelper(name string) bool {
	return slices.Contains(knownResponseHelperNames, name)
}

// discoverHelperFuncs scans all parsed files for package-level functions
// that accept *gin.Context as their first parameter. These are potential
// response helpers that can be traced for cross-function response extraction.
// Helpers are stored per-package to avoid cross-package name collisions.
func (a *HandlerAnalyzer) discoverHelperFuncs() {
	for _, file := range a.files {
		pkg := file.Name.Name
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
				continue
			}
			// Check if first parameter is *gin.Context
			firstParam := fn.Type.Params.List[0]
			if isGinContextType(firstParam.Type) {
				if a.helperFuncs[pkg] == nil {
					a.helperFuncs[pkg] = make(map[string]*ast.FuncDecl)
				}
				a.helperFuncs[pkg][fn.Name.Name] = fn
			}
		}
	}
	count := 0
	for _, funcs := range a.helperFuncs {
		count += len(funcs)
	}
	if count > 0 {
		slog.Debug("Discovered potential helper functions", "count", count)
	}
}

// pkgForFunc determines which package a function declaration belongs to
// by searching all parsed files for a pointer match.
func (a *HandlerAnalyzer) pkgForFunc(fn *ast.FuncDecl) string {
	for _, file := range a.files {
		for _, decl := range file.Decls {
			if f, ok := decl.(*ast.FuncDecl); ok && f == fn {
				return file.Name.Name
			}
		}
	}
	return ""
}

// lookupHelper finds a helper function by name, preferring the caller's package
// scope and falling back to a global search across all packages.
func (a *HandlerAnalyzer) lookupHelper(callerPkg, name string) *ast.FuncDecl {
	// Try caller's package first
	if pkgFuncs, ok := a.helperFuncs[callerPkg]; ok {
		if fn, ok := pkgFuncs[name]; ok {
			return fn
		}
	}
	// Fallback: search all packages
	for _, pkgFuncs := range a.helperFuncs {
		if fn, ok := pkgFuncs[name]; ok {
			return fn
		}
	}
	return nil
}

// parseHelperCall detects standalone helper function calls like done(c, data, err)
// and extracts response types from the arguments.
// Unlike parseHandlerCall which handles c.XXX() method calls, this handles
// bare function calls where the first arg is *gin.Context.
//
// Two detection strategies:
//  1. Known helpers (done, respond, etc.) — extract types from call arguments directly
//  2. Discovered helpers — trace into the function body to find c.JSON/c.XML calls
func (a *HandlerAnalyzer) parseHelperCall(call *ast.CallExpr, info *HandlerInfo, varTypeMap map[string]string) {
	// Only handle simple function calls: done(...), not c.XXX() or pkg.Fn()
	ident, ok := call.Fun.(*ast.Ident)
	if !ok {
		return
	}

	// Need at least 2 args: c + something
	if len(call.Args) < 2 {
		return
	}

	// Prefer traced helper declaration when available — the actual body gives
	// more accurate response types/statuses than the name-based heuristic.
	if helper := a.lookupHelper(a.callerPkg, ident.Name); helper != nil {
		a.traceHelperResponse(helper, call.Args, info, varTypeMap)
		return
	}

	// Fallback: known helper names without discovered implementation
	if isKnownResponseHelper(ident.Name) {
		a.extractHelperResponse(call.Args, info, varTypeMap)
	}
}

// traceHelperResponse traces into a helper function's body to find c.JSON/c.XML
// response calls and extracts response types by mapping call-site arguments to
// the helper's parameters.
func (a *HandlerAnalyzer) traceHelperResponse(helperDecl *ast.FuncDecl, callArgs []ast.Expr, info *HandlerInfo, varTypeMap map[string]string) {
	if helperDecl.Body == nil {
		return
	}

	// Build param-to-argument mapping for the helper function
	// e.g., helper(c *gin.Context, data any, err error) called as helper(c, project, nil)
	// maps: data → project (with type from varTypeMap), err → nil
	paramArgMap := a.buildParamArgMap(helperDecl, callArgs, varTypeMap)

	// Scan helper body for c.JSON/c.XML calls
	ast.Inspect(helperDecl.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if _, ok = sel.X.(*ast.Ident); !ok {
			return true
		}

		method := sel.Sel.Name
		switch method {
		case "JSON", "XML", "YAML":
			if len(call.Args) >= 2 {
				// Resolve the response argument through param mapping
				resolvedArg := a.resolveThroughParamMap(call.Args[1], paramArgMap)
				statusCode := extractStatusCode(call.Args[0])
				goType := extractTypeFromResponseWithStatus(resolvedArg, varTypeMap, statusCode)
				if goType != "" {
					info.Responses = append(info.Responses, ResponseInfo{
						StatusCode: statusCode,
						GoType:     goType,
					})
					slog.Debug("Traced helper response", "helper", helperDecl.Name.Name, "status", statusCode, "type", goType)
				}
			}
		}
		return true
	})
}

// buildParamArgMap maps helper function parameter names to their call-site arguments.
// e.g., for helper(c *gin.Context, data any, err error) called as helper(c, project, nil):
//
//	"data" → *ast.Ident{Name: "project"}, "err" → *ast.Ident{Name: "nil"}
func (a *HandlerAnalyzer) buildParamArgMap(helperDecl *ast.FuncDecl, callArgs []ast.Expr, _ map[string]string) map[string]ast.Expr {
	paramArgMap := make(map[string]ast.Expr)
	if helperDecl.Type.Params == nil {
		return paramArgMap
	}

	params := helperDecl.Type.Params.List
	for i, param := range params {
		if i >= len(callArgs) {
			break
		}
		// Skip the first param (*gin.Context) — it's always the context
		if i == 0 {
			continue
		}
		for _, name := range param.Names {
			paramArgMap[name.Name] = callArgs[i]
		}
	}
	return paramArgMap
}

// resolveThroughParamMap resolves an expression by substituting helper parameters
// with their call-site arguments. Returns new AST nodes instead of mutating originals
// to avoid corrupting shared helper declarations across multiple handler analyses.
func (a *HandlerAnalyzer) resolveThroughParamMap(expr ast.Expr, paramToArg map[string]ast.Expr) ast.Expr {
	switch e := expr.(type) {
	case *ast.Ident:
		if arg, exists := paramToArg[e.Name]; exists {
			return arg
		}
	case *ast.CompositeLit:
		newElts := make([]ast.Expr, len(e.Elts))
		for i, elt := range e.Elts {
			newElts[i] = a.resolveThroughParamMap(elt, paramToArg)
		}
		return &ast.CompositeLit{
			Type:   e.Type,
			Lbrace: e.Lbrace,
			Elts:   newElts,
			Rbrace: e.Rbrace,
		}
	case *ast.KeyValueExpr:
		return &ast.KeyValueExpr{
			Key:   e.Key,
			Colon: e.Colon,
			Value: a.resolveThroughParamMap(e.Value, paramToArg),
		}
	}
	return expr
}
