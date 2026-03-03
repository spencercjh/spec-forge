package spring

import (
	"fmt"
	"os"
	"strings"
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
	return os.WriteFile(path, content, 0644)
}

// insertAfter inserts a line after the target line in content.
func insertAfter(content, target, newLine string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inserted := false

	for _, line := range lines {
		result = append(result, line)
		if !inserted && strings.Contains(line, target) {
			// Calculate indentation from the next line if available
			indent := ""
			if len(result) > 1 {
				prevLine := result[len(result)-2]
				for _, c := range prevLine {
					if c == ' ' || c == '\t' {
						indent += string(c)
					} else {
						break
					}
				}
				// Add one level of indentation
				indent += "    "
			}
			result = append(result, indent+newLine)
			inserted = true
		}
	}

	return strings.Join(result, "\n")
}
