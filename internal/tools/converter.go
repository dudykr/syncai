package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dudykr/syncai/internal/types"
	"github.com/sirupsen/logrus"
)

// Converter handles conversion of cursor rules to different AI tool formats
type Converter struct {
	logger    *logrus.Logger
	outputDir string
}

// NewConverter creates a new converter instance
func NewConverter(outputDir string, logger *logrus.Logger) *Converter {
	return &Converter{
		outputDir: outputDir,
		logger:    logger,
	}
}

// ConvertRules converts cursor rules to specified target tools
func (c *Converter) ConvertRules(rules *types.CursorRules, targets []types.TargetTool) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target tools specified")
	}

	toolConfigs := types.GetToolConfigs()
	var wg sync.WaitGroup
	errChan := make(chan error, len(targets))

	// Convert to each target tool in parallel
	for _, target := range targets {
		wg.Add(1)
		go func(target types.TargetTool) {
			defer wg.Done()

			config, exists := toolConfigs[target]
			if !exists {
				errChan <- fmt.Errorf("unsupported target tool: %s", target)
				return
			}

			c.logger.Infof("Converting rules for %s", target)
			if err := c.convertToTool(rules, config); err != nil {
				errChan <- fmt.Errorf("failed to convert to %s: %w", target, err)
				return
			}
			c.logger.Infof("Successfully converted rules for %s", target)
		}(target)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	var errors []string
	for err := range errChan {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		return fmt.Errorf("conversion errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// convertToTool converts rules to a specific tool format
func (c *Converter) convertToTool(rules *types.CursorRules, config types.ToolConfig) error {
	// Use outputDir directly instead of creating tool-specific directories
	targetDir := c.outputDir

	switch config.Tool {
	case types.TargetCursor:
		return c.convertToCursor(rules, config, targetDir)
	case types.TargetWindSurf:
		return c.convertToWindSurf(rules, config, targetDir)
	case types.TargetRooCode:
		return c.convertToRooCode(rules, config, targetDir)
	case types.TargetCline:
		return c.convertToCline(rules, config, targetDir)
	default:
		return fmt.Errorf("unsupported tool: %s", config.Tool)
	}
}

// convertToCursor converts to Cursor IDE format (essentially a copy)
func (c *Converter) convertToCursor(rules *types.CursorRules, config types.ToolConfig, targetDir string) error {
	// Write global rules
	if rules.GlobalRules != "" {
		globalPath := filepath.Join(targetDir, config.ConfigPath)
		if err := c.writeFile(globalPath, rules.GlobalRules); err != nil {
			return err
		}
	}

	// Write folder rules
	for folderPath, content := range rules.FolderRules {
		folderRulesDir := filepath.Join(targetDir, folderPath, ".cursor", "rules")
		if err := os.MkdirAll(folderRulesDir, 0755); err != nil {
			return err
		}

		rulePath := filepath.Join(folderRulesDir, "rules")
		if err := c.writeFile(rulePath, content); err != nil {
			return err
		}
	}

	// Write MDC rules
	for _, mdcRule := range rules.MDCRules {
		// Recreate the original path structure
		relPath, _ := filepath.Rel(c.outputDir, mdcRule.FilePath)
		mdcPath := filepath.Join(targetDir, relPath)

		if err := os.MkdirAll(filepath.Dir(mdcPath), 0755); err != nil {
			return err
		}

		content := c.buildMDCContent(mdcRule)
		if err := c.writeFile(mdcPath, content); err != nil {
			return err
		}
	}

	return nil
}

// convertToWindSurf converts to WindSurf format
func (c *Converter) convertToWindSurf(rules *types.CursorRules, config types.ToolConfig, targetDir string) error {
	// WindSurf only supports global rules
	content := c.buildGlobalContent(rules)

	globalPath := filepath.Join(targetDir, config.ConfigPath)
	return c.writeFile(globalPath, content)
}

// convertToRooCode converts to Roo Code format
func (c *Converter) convertToRooCode(rules *types.CursorRules, config types.ToolConfig, targetDir string) error {
	// Convert targetDir to absolute path for proper relative path calculation
	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for target directory: %w", err)
	}

	// Write global rules to the root .roo/rules directory
	if rules.GlobalRules != "" {
		rooRulesDir := filepath.Join(targetDir, ".roo", "rules")
		if err := os.MkdirAll(rooRulesDir, 0755); err != nil {
			return fmt.Errorf("failed to create .roo/rules directory: %w", err)
		}

		globalRulesPath := filepath.Join(rooRulesDir, "01-global.md")
		globalContent := "# Global Rules\n\n" + rules.GlobalRules
		if err := c.writeFile(globalRulesPath, globalContent); err != nil {
			return err
		}
	}

	// Write MDC rules to their respective .roo directories (same level as .cursor)
	for _, mdcRule := range rules.MDCRules {
		// Get the folder path from the MDC file path
		// mdcRule.FilePath is like "/path/to/project/frontend/.cursor/rules/testing.mdc"
		// We want to extract "frontend" part
		relPath, err := filepath.Rel(absTargetDir, mdcRule.FilePath)
		if err != nil {
			// If we can't get relative path, skip this rule or put in root
			c.logger.Warnf("Could not determine relative path for MDC rule %s: %v", mdcRule.Name, err)
			continue
		}

		// Extract folder path from the relative path
		// relPath is like "frontend/.cursor/rules/testing.mdc"
		// We want "frontend"
		parts := strings.Split(relPath, string(filepath.Separator))

		if len(parts) < 3 || parts[1] != ".cursor" || parts[2] != "rules" {
			// This MDC file is not in a standard .cursor/rules structure, put in root
			rooRulesDir := filepath.Join(targetDir, ".roo", "rules")
			if err := os.MkdirAll(rooRulesDir, 0755); err != nil {
				return fmt.Errorf("failed to create .roo/rules directory: %w", err)
			}

			filename := fmt.Sprintf("%s.md", sanitizeFilename(mdcRule.Name))
			rulePath := filepath.Join(rooRulesDir, filename)
			content := fmt.Sprintf("# %s\n\n%s", mdcRule.Name, mdcRule.Content)
			if err := c.writeFile(rulePath, content); err != nil {
				return err
			}
			continue
		}

		// Create .roo directory at the same level as .cursor
		folderPath := parts[0]
		folderRooDir := filepath.Join(targetDir, folderPath, ".roo")
		if err := os.MkdirAll(folderRooDir, 0755); err != nil {
			return fmt.Errorf("failed to create .roo directory for %s: %w", folderPath, err)
		}

		// Write MDC rule to folder's .roo directory
		filename := fmt.Sprintf("%s.md", sanitizeFilename(mdcRule.Name))
		rulePath := filepath.Join(folderRooDir, filename)

		content := fmt.Sprintf("# %s\n\n%s", mdcRule.Name, mdcRule.Content)
		if len(mdcRule.Globs) > 0 {
			content = fmt.Sprintf("# %s\n\n**Applies to:** %s\n\n%s", mdcRule.Name, strings.Join(mdcRule.Globs, ", "), mdcRule.Content)
		}

		if err := c.writeFile(rulePath, content); err != nil {
			return err
		}
	}

	// Write folder rules to their respective .roo directories (same level as .cursor)
	for folderPath, folderContent := range rules.FolderRules {
		// Create .roo directory at the same level as .cursor
		folderRooDir := filepath.Join(targetDir, folderPath, ".roo")
		if err := os.MkdirAll(folderRooDir, 0755); err != nil {
			return fmt.Errorf("failed to create .roo directory for %s: %w", folderPath, err)
		}

		// Write folder-specific rules to .roo/rules.md
		rulePath := filepath.Join(folderRooDir, "rules.md")
		content := fmt.Sprintf("# Rules for %s\n\n%s", folderPath, folderContent)
		if err := c.writeFile(rulePath, content); err != nil {
			return err
		}
	}

	return nil
}

// sanitizeFilename removes invalid characters from filename
func sanitizeFilename(name string) string {
	// Replace spaces and invalid characters with hyphens
	result := strings.ToLower(name)
	result = strings.ReplaceAll(result, " ", "-")
	result = strings.ReplaceAll(result, "/", "-")
	result = strings.ReplaceAll(result, "\\", "-")
	result = strings.ReplaceAll(result, ":", "-")
	result = strings.ReplaceAll(result, "*", "-")
	result = strings.ReplaceAll(result, "?", "-")
	result = strings.ReplaceAll(result, "\"", "-")
	result = strings.ReplaceAll(result, "<", "-")
	result = strings.ReplaceAll(result, ">", "-")
	result = strings.ReplaceAll(result, "|", "-")

	// Remove multiple consecutive hyphens
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}

	// Trim hyphens from start and end
	result = strings.Trim(result, "-")

	// If empty, use default name
	if result == "" {
		result = "unnamed"
	}

	return result
}

// convertToCline converts to Cline format
func (c *Converter) convertToCline(rules *types.CursorRules, config types.ToolConfig, targetDir string) error {
	content := c.buildGlobalContent(rules)

	// Create .clinerules file
	clinerulePath := filepath.Join(targetDir, config.ConfigPath)
	if err := c.writeFile(clinerulePath, content); err != nil {
		return err
	}

	// Create .cline directory with instructions.md
	clineDir := filepath.Join(targetDir, ".cline")
	if err := os.MkdirAll(clineDir, 0755); err != nil {
		return fmt.Errorf("failed to create .cline directory: %w", err)
	}

	instructionsPath := filepath.Join(clineDir, "instructions.md")
	return c.writeFile(instructionsPath, content)
}

// buildGlobalContent builds the global content combining all rules
func (c *Converter) buildGlobalContent(rules *types.CursorRules) string {
	var parts []string

	// Add global rules
	if rules.GlobalRules != "" {
		parts = append(parts, "# Global Rules\n\n"+rules.GlobalRules)
	}

	// Add always-apply MDC rules
	for _, mdcRule := range rules.MDCRules {
		if mdcRule.AlwaysApply {
			parts = append(parts, fmt.Sprintf("# %s\n\n%s", mdcRule.Name, mdcRule.Content))
		}
	}

	// Add folder rules as context
	if len(rules.FolderRules) > 0 {
		parts = append(parts, "\n# Folder-specific Rules\n")
		for folderPath, content := range rules.FolderRules {
			parts = append(parts, fmt.Sprintf("## Rules for %s\n\n%s", folderPath, content))
		}
	}

	// Add conditional MDC rules as context
	conditionalRules := []types.MDCRule{}
	for _, mdcRule := range rules.MDCRules {
		if !mdcRule.AlwaysApply {
			conditionalRules = append(conditionalRules, mdcRule)
		}
	}

	if len(conditionalRules) > 0 {
		parts = append(parts, "\n# Conditional Rules\n")
		for _, mdcRule := range conditionalRules {
			globsInfo := ""
			if len(mdcRule.Globs) > 0 {
				globsInfo = fmt.Sprintf(" (applies to: %s)", strings.Join(mdcRule.Globs, ", "))
			}
			parts = append(parts, fmt.Sprintf("## %s%s\n\n%s", mdcRule.Name, globsInfo, mdcRule.Content))
		}
	}

	return strings.Join(parts, "\n\n")
}

// buildMDCContent builds the MDC content with frontmatter
func (c *Converter) buildMDCContent(rule types.MDCRule) string {
	var parts []string

	// Add frontmatter if needed
	if rule.Description != "" || rule.AlwaysApply || len(rule.Globs) > 0 {
		parts = append(parts, "---")

		if rule.Name != "" {
			parts = append(parts, fmt.Sprintf("name: %s", rule.Name))
		}
		if rule.Description != "" {
			parts = append(parts, fmt.Sprintf("description: %s", rule.Description))
		}
		if rule.AlwaysApply {
			parts = append(parts, "alwaysApply: true")
		}
		if len(rule.Globs) > 0 {
			parts = append(parts, "globs:")
			for _, glob := range rule.Globs {
				parts = append(parts, fmt.Sprintf("  - %s", glob))
			}
		}

		parts = append(parts, "---", "")
	}

	parts = append(parts, rule.Content)
	return strings.Join(parts, "\n")
}

// writeFile writes content to a file
func (c *Converter) writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(content), 0644)
}
