package gin

import "testing"

func TestInfo(t *testing.T) {
	info := Info{
		GoVersion:  "1.21",
		ModuleName: "github.com/example/app",
		GinVersion: "v1.9.1",
		HasGin:     true,
	}

	if info.GoVersion != "1.21" {
		t.Errorf("expected GoVersion '1.21', got %s", info.GoVersion)
	}
	if info.ModuleName != "github.com/example/app" {
		t.Errorf("expected ModuleName 'github.com/example/app', got %s", info.ModuleName)
	}
}

func TestRouterGroup(t *testing.T) {
	rg := RouterGroup{
		BasePath: "/api/v1",
		Routes: []Route{
			{Method: "GET", Path: "/users"},
			{Method: "POST", Path: "/users"},
		},
	}

	if rg.BasePath != "/api/v1" {
		t.Errorf("expected BasePath '/api/v1', got %s", rg.BasePath)
	}
	if len(rg.Routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(rg.Routes))
	}
}

func TestRoute(t *testing.T) {
	route := Route{
		Method:      "GET",
		Path:        "/users/:id",
		FullPath:    "/api/v1/users/:id",
		HandlerName: "GetUser",
		HandlerFile: "user_handler.go",
		Middlewares: []string{"Auth"},
	}

	if route.Method != "GET" {
		t.Errorf("expected Method 'GET', got %s", route.Method)
	}
	if route.FullPath != "/api/v1/users/:id" {
		t.Errorf("expected FullPath '/api/v1/users/:id', got %s", route.FullPath)
	}
}

func TestHandlerInfo(t *testing.T) {
	hi := HandlerInfo{
		PathParams: []ParamInfo{
			{Name: "id", GoType: "string", Required: true},
		},
		QueryParams: []ParamInfo{
			{Name: "page", GoType: "string", Required: false},
		},
		BodyType: "CreateUserRequest",
		Responses: []ResponseInfo{
			{StatusCode: 200, GoType: "User"},
			{StatusCode: 404, GoType: "ErrorResponse"},
		},
	}

	if len(hi.PathParams) != 1 {
		t.Errorf("expected 1 path param, got %d", len(hi.PathParams))
	}
	if len(hi.Responses) != 2 {
		t.Errorf("expected 2 responses, got %d", len(hi.Responses))
	}
}

func TestParamInfo(t *testing.T) {
	param := ParamInfo{
		Name:     "id",
		GoType:   "string",
		Required: true,
	}

	if param.Name != "id" {
		t.Errorf("expected Name 'id', got %s", param.Name)
	}
	if !param.Required {
		t.Error("expected Required to be true")
	}
}

func TestResponseInfo(t *testing.T) {
	resp := ResponseInfo{
		StatusCode: 200,
		GoType:     "User",
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected StatusCode 200, got %d", resp.StatusCode)
	}
	if resp.GoType != "User" {
		t.Errorf("expected GoType 'User', got %s", resp.GoType)
	}
}
