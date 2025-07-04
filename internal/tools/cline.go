package tools

import (
	"encoding/json"
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
	
	// Cline uses .vscode/settings.json with cline.customInstructions
	vscodeDir := filepath.Join(config.RootPath, ".vscode")
	settingsPath := filepath.Join(vscodeDir, "settings.json")
	
	// Create .vscode directory if it doesn't exist
	if err := os.MkdirAll(vscodeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .vscode directory: %w", err)
	}
	
	// Read existing settings.json if it exists
	var settings map[string]interface{}
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			settings = make(map[string]interface{})
		}
	} else {
		settings = make(map[string]interface{})
	}
	
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
	
	// Set the custom instructions
	settings["cline.customInstructions"] = instructions.String()
	
	// Write settings.json
	settingsData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}
	
	err = os.WriteFile(settingsPath, settingsData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write settings.json: %w", err)
	}
	
	fmt.Printf("  ✓ Updated .vscode/settings.json with cline.customInstructions\n")
	return nil
}

func (c *Cline) Import(rootPath string) (*ProjectConfig, error) {
	config := &ProjectConfig{
		RootPath: rootPath,
	}
	
	// Read from .vscode/settings.json
	settingsPath := filepath.Join(rootPath, ".vscode", "settings.json")
	if data, err := os.ReadFile(settingsPath); err == nil {
		var settings map[string]interface{}
		if err := json.Unmarshal(data, &settings); err == nil {
			if instructions, ok := settings["cline.customInstructions"].(string); ok {
				config.CursorRules = instructions
			}
		}
	}
	
	return config, nil
}