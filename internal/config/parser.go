package config

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dudykr/syncai/internal/types"
	"gopkg.in/yaml.v3"
)

// Parser handles parsing of cursor rules files
type Parser struct {
	rootDir string
}

// NewParser creates a new parser instance
func NewParser(rootDir string) *Parser {
	return &Parser{rootDir: rootDir}
}

// ParseCursorRules parses all cursor rules files in the project
func (p *Parser) ParseCursorRules() (*types.CursorRules, error) {
	rules := &types.CursorRules{
		FolderRules: make(map[string]string),
		MDCRules:    []types.MDCRule{},
	}

	// Parse global .cursorrules file
	globalRulesPath := filepath.Join(p.rootDir, ".cursorrules")
	if content, err := p.readFileIfExists(globalRulesPath); err == nil {
		rules.GlobalRules = content
	}

	// Find and parse all .cursor/rules directories
	err := p.walkCursorDirs(rules)
	if err != nil {
		return nil, fmt.Errorf("failed to walk cursor directories: %w", err)
	}

	return rules, nil
}

// walkCursorDirs walks through all .cursor directories and parses rules
func (p *Parser) walkCursorDirs(rules *types.CursorRules) error {
	return filepath.WalkDir(p.rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		// Check if this is a .cursor directory
		if d.Name() == ".cursor" {
			rulesDir := filepath.Join(path, "rules")
			if _, err := os.Stat(rulesDir); err == nil {
				return p.parseRulesDir(rulesDir, rules)
			}
		}

		return nil
	})
}

// parseRulesDir parses a .cursor/rules directory
func (p *Parser) parseRulesDir(rulesDir string, rules *types.CursorRules) error {
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(rulesDir, entry.Name())

		// Handle .mdc files
		if strings.HasSuffix(entry.Name(), ".mdc") {
			mdcRule, err := p.parseMDCFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to parse MDC file %s: %w", filePath, err)
			}
			rules.MDCRules = append(rules.MDCRules, *mdcRule)
		} else {
			// Handle regular rule files as folder rules
			content, err := p.readFileIfExists(filePath)
			if err != nil {
				return err
			}

			// Get relative path from project root for the folder rule
			// rulesDir is .../somefolder/.cursor/rules, we want .../somefolder
			cursorParentDir := filepath.Dir(filepath.Dir(rulesDir))
			relPath, err := filepath.Rel(p.rootDir, cursorParentDir)
			if err != nil {
				relPath = cursorParentDir
			}

			rules.FolderRules[relPath] = content
		}
	}

	return nil
}

// parseMDCFile parses a .mdc file and extracts metadata and content
func (p *Parser) parseMDCFile(filePath string) (*types.MDCRule, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	rule := &types.MDCRule{
		FilePath: filePath,
		Name:     strings.TrimSuffix(filepath.Base(filePath), ".mdc"),
	}

	contentStr := string(content)

	// Parse frontmatter if exists
	if strings.HasPrefix(contentStr, "---") {
		parts := strings.SplitN(contentStr, "---", 3)
		if len(parts) >= 3 {
			// Parse YAML frontmatter
			var frontmatter map[string]interface{}
			if err := yaml.Unmarshal([]byte(parts[1]), &frontmatter); err == nil {
				if name, ok := frontmatter["name"].(string); ok {
					rule.Name = name
				}
				if desc, ok := frontmatter["description"].(string); ok {
					rule.Description = desc
				}
				if alwaysApply, ok := frontmatter["alwaysApply"].(bool); ok {
					rule.AlwaysApply = alwaysApply
				}
				if globs, ok := frontmatter["globs"].([]interface{}); ok {
					for _, glob := range globs {
						if globStr, ok := glob.(string); ok {
							rule.Globs = append(rule.Globs, globStr)
						}
					}
				}
			}

			// Content is everything after the second ---
			rule.Content = strings.TrimSpace(parts[2])
		} else {
			rule.Content = contentStr
		}
	} else {
		// No frontmatter, entire content is the rule
		rule.Content = contentStr

		// Try to extract metadata from comments
		rule.AlwaysApply, rule.Globs = p.extractMetadataFromComments(contentStr)
	}

	return rule, nil
}

// extractMetadataFromComments extracts alwaysApply and globs from comments
func (p *Parser) extractMetadataFromComments(content string) (bool, []string) {
	alwaysApply := false
	var globs []string

	// Look for comments that might contain metadata
	commentRegex := regexp.MustCompile(`(?m)^//\s*@(\w+):\s*(.+)$`)
	matches := commentRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			key := match[1]
			value := strings.TrimSpace(match[2])

			switch key {
			case "alwaysApply":
				alwaysApply = strings.ToLower(value) == "true"
			case "globs":
				// Parse comma-separated globs
				globList := strings.Split(value, ",")
				for _, glob := range globList {
					globs = append(globs, strings.TrimSpace(glob))
				}
			}
		}
	}

	return alwaysApply, globs
}

// readFileIfExists reads a file if it exists, returns empty string if not
func (p *Parser) readFileIfExists(path string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
