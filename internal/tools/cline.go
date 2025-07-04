package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Cline struct{}

func (c *Cline) Name() string {
	return "cline"
}

func (c *Cline) Build(config *ProjectConfig) error {
	fmt.Printf("Building Cline configuration...\n")
	
	// Cline uses .clinerules file
	clinerrulesPath := filepath.Join(config.RootPath, ".clinerules")
	
	// Build custom instructions
	var instructions strings.Builder
	
	// Add global rules from .cursorrules
	if config.CursorRules != "" {
		instructions.WriteString("# Global Instructions\n\n")
		instructions.WriteString(config.CursorRules)
		instructions.WriteString("\n\n")
	}
	
	// Add MDC files content
	if len(config.MdcFiles) > 0 {
		instructions.WriteString("# Context-specific Instructions\n\n")
		for _, mdcFile := range config.MdcFiles {
			if mdcFile.Description != "" {
				instructions.WriteString(fmt.Sprintf("## %s\n", mdcFile.Description))
			}
			if len(mdcFile.Globs) > 0 {
				instructions.WriteString(fmt.Sprintf("**File Patterns:** %s\n", strings.Join(mdcFile.Globs, ", ")))
			}
			if mdcFile.AlwaysApply {
				instructions.WriteString("**Always Apply:** Yes\n")
			}
			instructions.WriteString("\n")
			instructions.WriteString(mdcFile.Content)
			instructions.WriteString("\n\n")
		}
	}
	
	if instructions.Len() == 0 {
		fmt.Printf("  ⚠ No rules found to generate Cline configuration\n")
		return nil
	}
	
	// Write .clinerules file
	err := os.WriteFile(clinerrulesPath, []byte(instructions.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write .clinerules: %w", err)
	}
	
	fmt.Printf("  ✓ Updated .clinerules\n")
	return nil
}

func (c *Cline) Import(rootPath string) (*ProjectConfig, error) {
	config := &ProjectConfig{
		RootPath: rootPath,
	}
	
	// Read from .clinerules
	clinerrulesPath := filepath.Join(rootPath, ".clinerules")
	if data, err := os.ReadFile(clinerrulesPath); err == nil {
		config.CursorRules = string(data)
	}
	
	return config, nil
}