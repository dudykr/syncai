package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type RooCode struct{}

func (r *RooCode) Name() string {
	return "roo-code"
}

func (r *RooCode) Build(config *ProjectConfig) error {
	fmt.Printf("Building Roo Code configuration...\n")
	
	// Roo Code uses .roocode directory with context files
	roocodeDir := filepath.Join(config.RootPath, ".roocode")
	
	// Create .roocode directory if it doesn't exist
	if err := os.MkdirAll(roocodeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .roocode directory: %w", err)
	}
	
	// Create global context file
	if config.CursorRules != "" {
		globalContextPath := filepath.Join(roocodeDir, "global.md")
		err := os.WriteFile(globalContextPath, []byte("# Global Context\n\n"+config.CursorRules), 0644)
		if err != nil {
			return fmt.Errorf("failed to write global context: %w", err)
		}
		fmt.Printf("  ✓ Generated .roocode/global.md\n")
	}
	
	// Create context files for each MDC file
	for i, mdcFile := range config.MdcFiles {
		contextFile := fmt.Sprintf("context_%d.md", i+1)
		if mdcFile.Description != "" {
			// Use description as filename (sanitized)
			contextFile = fmt.Sprintf("%s.md", sanitizeFilename(mdcFile.Description))
		}
		
		contextPath := filepath.Join(roocodeDir, contextFile)
		
		var content strings.Builder
		if mdcFile.Description != "" {
			content.WriteString(fmt.Sprintf("# %s\n\n", mdcFile.Description))
		}
		
		if len(mdcFile.Globs) > 0 {
			content.WriteString("## File Patterns\n")
			for _, glob := range mdcFile.Globs {
				content.WriteString(fmt.Sprintf("- %s\n", glob))
			}
			content.WriteString("\n")
		}
		
		if mdcFile.AlwaysApply {
			content.WriteString("**Always Apply:** Yes\n\n")
		}
		
		content.WriteString(mdcFile.Content)
		
		err := os.WriteFile(contextPath, []byte(content.String()), 0644)
		if err != nil {
			return fmt.Errorf("failed to write context file %s: %w", contextFile, err)
		}
		
		fmt.Printf("  ✓ Generated .roocode/%s\n", contextFile)
	}
	
	if config.CursorRules == "" && len(config.MdcFiles) == 0 {
		fmt.Printf("  ⚠ No rules found to generate Roo Code configuration\n")
	}
	
	return nil
}

func (r *RooCode) Import(rootPath string) (*ProjectConfig, error) {
	config := &ProjectConfig{
		RootPath: rootPath,
	}
	
	// Read all .md files from .roocode directory
	roocodeDir := filepath.Join(rootPath, ".roocode")
	if _, err := os.Stat(roocodeDir); os.IsNotExist(err) {
		return config, nil
	}
	
	var allContent strings.Builder
	
	err := filepath.Walk(roocodeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			allContent.WriteString(string(data))
			allContent.WriteString("\n\n")
		}
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to read .roocode directory: %w", err)
	}
	
	config.CursorRules = allContent.String()
	return config, nil
}

func sanitizeFilename(filename string) string {
	// Replace invalid characters with underscores
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	result := filename
	for _, char := range invalidChars {
		result = strings.ReplaceAll(result, char, "_")
	}
	return result
}