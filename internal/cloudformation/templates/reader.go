package templates

import (
	"embed"
	"fmt"
)

//go:embed *.yaml
var templateFS embed.FS

// Read reads a CloudFormation template file from embedded templates
func Read(filename string) (string, error) {
	data, err := templateFS.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read template %s: %w", filename, err)
	}
	return string(data), nil
}
