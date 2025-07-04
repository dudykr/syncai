package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type WindSurf struct{}

func (w *WindSurf) Name() string {
	return "windsurf"
}

func (w *WindSurf) Build(config *ProjectConfig) error {
	fmt.Printf("Building WindSurf configuration...\n")
	
	// WindSurf uses .windsurfrules file
	windsurfRulesPath := filepath.Join(config.RootPath, ".windsurfrules")
	
	var content strings.Builder
	
	// Add global rules from .cursorrules
	if config.CursorRules != "" {
		content.WriteString("# Global Rules\n")
		content.WriteString(config.CursorRules)
		content.WriteString("\n\n")
	}
	
	// Add MDC files content
	if len(config.MdcFiles) > 0 {
		content.WriteString("# Context-specific Rules\n\n")
		for _, mdcFile := range config.MdcFiles {
			if mdcFile.Description != "" {
				content.WriteString(fmt.Sprintf("## %s\n", mdcFile.Description))
			}
			if len(mdcFile.Globs) > 0 {
				content.WriteString(fmt.Sprintf("**Applies to:** %s\n", strings.Join(mdcFile.Globs, ", ")))
			}
			if mdcFile.AlwaysApply {
				content.WriteString("**Always Apply:** Yes\n")
			}
			content.WriteString("\n")
			content.WriteString(mdcFile.Content)
			content.WriteString("\n\n")
		}
	}
	
	if content.Len() == 0 {
		fmt.Printf("  ⚠ No rules found to generate WindSurf configuration\n")
		return nil
	}
	
	err := os.WriteFile(windsurfRulesPath, []byte(content.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write .windsurfrules: %w", err)
	}
	
	fmt.Printf("  ✓ Generated .windsurfrules\n")
	return nil
}

func (w *WindSurf) Import(rootPath string) (*ProjectConfig, error) {
	config := &ProjectConfig{
		RootPath: rootPath,
	}
	
	// WindSurf uses .windsurfrules file
	windsurfRulesPath := filepath.Join(rootPath, ".windsurfrules")
	if data, err := os.ReadFile(windsurfRulesPath); err == nil {
		config.CursorRules = string(data)
	}
	
	return config, nil
}