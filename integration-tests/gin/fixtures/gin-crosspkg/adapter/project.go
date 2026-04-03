package adapter

import "context"

// CreateProject creates a project in the external system.
// Same name as the handler in package apis, but no *gin.Context.
func CreateProject(ctx context.Context, name string, quota float64) error {
	return nil
}

// GetProject retrieves a project from the external system.
// Same name as the handler in package apis, but no *gin.Context.
func GetProject(ctx context.Context, name string) (string, error) {
	return "", nil
}

// UpdateProject updates a project in the external system.
// Same name as the handler in package apis, but no *gin.Context.
func UpdateProject(ctx context.Context, name string, quota float64) error {
	return nil
}

// DeleteProject deletes a project from the external system.
// Same name as the handler in package apis, but no *gin.Context.
func DeleteProject(ctx context.Context, name string) error {
	return nil
}
