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
	targetDir := filepath.Join(c.outputDir, string(config.Tool))
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

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
	// Write global rules
	content := c.buildGlobalContent(rules)
	globalPath := filepath.Join(targetDir, config.ConfigPath)
	if err := c.writeFile(globalPath, content); err != nil {
		return err
	}

	// Write folder rules
	for folderPath, folderContent := range rules.FolderRules {
		folderRulesPath := filepath.Join(targetDir, folderPath, config.FolderConfigName)
		if err := os.MkdirAll(filepath.Dir(folderRulesPath), 0755); err != nil {
			return err
		}

		if err := c.writeFile(folderRulesPath, folderContent); err != nil {
			return err
		}
	}

	return nil
}

// convertToCline converts to Cline format
func (c *Converter) convertToCline(rules *types.CursorRules, config types.ToolConfig, targetDir string) error {
	// Cline only supports global rules
	content := c.buildGlobalContent(rules)

	globalPath := filepath.Join(targetDir, config.ConfigPath)
	if err := os.MkdirAll(filepath.Dir(globalPath), 0755); err != nil {
		return err
	}

	return c.writeFile(globalPath, content)
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
