package spring

import (
	"fmt"
	"os"
)

// readFile reads file content as string.
func readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(content), nil
}

// writeFile writes content to file.
func writeFile(path string, content []byte) error {
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}
