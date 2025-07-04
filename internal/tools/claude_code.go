package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ClaudeCode struct{}

func (c *ClaudeCode) Name() string {
	return "claude-code"
}

func (c *ClaudeCode) Build(config *ProjectConfig) error {
	fmt.Printf("Building Claude Code configuration...\n")
	
	// Claude Code uses CLAUDE.md file
	claudeMdPath := filepath.Join(config.RootPath, "CLAUDE.md")
	
	var content strings.Builder
	
	// Add header
	content.WriteString("# Claude Code Instructions\n\n")
	content.WriteString("This file contains custom instructions for Claude Code.\n\n")
	
	// Add global rules from .cursorrules
	if config.CursorRules != "" {
		content.WriteString("## Global Instructions\n\n")
		content.WriteString(config.CursorRules)
		content.WriteString("\n\n")
	}
	
	// Add MDC files content
	if len(config.MdcFiles) > 0 {
		content.WriteString("## Context-specific Instructions\n\n")
		for _, mdcFile := range config.MdcFiles {
			if mdcFile.Description != "" {
				content.WriteString(fmt.Sprintf("### %s\n", mdcFile.Description))
			}
			if len(mdcFile.Globs) > 0 {
				content.WriteString(fmt.Sprintf("**File Patterns:** %s\n", strings.Join(mdcFile.Globs, ", ")))
			}
			if mdcFile.AlwaysApply {
				content.WriteString("**Always Apply:** Yes\n")
			}
			content.WriteString("\n")
			content.WriteString(mdcFile.Content)
			content.WriteString("\n\n")
		}
	}
	
	if config.CursorRules == "" && len(config.MdcFiles) == 0 {
		fmt.Printf("  ⚠ No rules found to generate Claude Code configuration\n")
		return nil
	}
	
	err := os.WriteFile(claudeMdPath, []byte(content.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write CLAUDE.md: %w", err)
	}
	
	fmt.Printf("  ✓ Generated CLAUDE.md\n")
	return nil
}

func (c *ClaudeCode) Import(rootPath string) (*ProjectConfig, error) {
	config := &ProjectConfig{
		RootPath: rootPath,
	}
	
	// Read from CLAUDE.md
	claudeMdPath := filepath.Join(rootPath, "CLAUDE.md")
	if data, err := os.ReadFile(claudeMdPath); err == nil {
		config.CursorRules = string(data)
	}
	
	return config, nil
}