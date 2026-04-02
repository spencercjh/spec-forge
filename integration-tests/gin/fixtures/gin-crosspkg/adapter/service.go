package adapter

import "context"

// Service is an external service client.
type Service struct{}

// CreateProject is a method with the same name as the adapter function and the handler.
// The receiver makes it a method, not a package-level function.
func (s *Service) CreateProject(ctx context.Context, name string) error {
	return nil
}

// GetProject is a method with the same name as the handler.
func (s *Service) GetProject(ctx context.Context, name string) (string, error) {
	return "", nil
}

// UpdateProject is a method with the same name as the handler.
func (s *Service) UpdateProject(ctx context.Context, name string) error {
	return nil
}

// DeleteProject is a method with the same name as the handler.
func (s *Service) DeleteProject(ctx context.Context, name string) error {
	return nil
}
