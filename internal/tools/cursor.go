package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Cursor struct{}

func (c *Cursor) Name() string {
	return "cursor"
}

func (c *Cursor) Build(config *ProjectConfig) error {
	fmt.Printf("Building Cursor configuration...\n")
	
	// Cursor already uses .cursorrules and .cursor/rules/*.mdc files
	// So we don't need to generate anything - just validate
	
	if config.CursorRules != "" {
		fmt.Printf("  ✓ .cursorrules file found\n")
	}
	
	if len(config.MdcFiles) > 0 {
		fmt.Printf("  ✓ %d MDC rule files found\n", len(config.MdcFiles))
	}
	
	return nil
}

func (c *Cursor) Import(rootPath string) (*ProjectConfig, error) {
	// For Cursor, we just read the existing files
	config := &ProjectConfig{
		RootPath: rootPath,
	}
	
	// Load .cursorrules file
	cursorRulesPath := filepath.Join(rootPath, ".cursorrules")
	if data, err := os.ReadFile(cursorRulesPath); err == nil {
		config.CursorRules = string(data)
	}
	
	// Find .cursor directories and load MDC files
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == ".cursor" {
			config.CursorDirs = append(config.CursorDirs, path)
		}
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to find .cursor directories: %w", err)
	}
	
	// Load MDC files
	for _, cursorDir := range config.CursorDirs {
		rulesDir := filepath.Join(cursorDir, "rules")
		if _, err := os.Stat(rulesDir); os.IsNotExist(err) {
			continue
		}
		
		err = filepath.Walk(rulesDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, ".mdc") {
				mdcFile, err := parseMdcFile(path)
				if err != nil {
					return err
				}
				config.MdcFiles = append(config.MdcFiles, *mdcFile)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to walk rules directory %s: %w", rulesDir, err)
		}
	}
	
	return config, nil
}
