package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/dudykr/syncai/internal/config"
	"github.com/dudykr/syncai/internal/tools"
	"github.com/dudykr/syncai/internal/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	logger    *logrus.Logger
	outputDir string
	verbose   bool
)

func main() {
	logger = logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	rootCmd := &cobra.Command{
		Use:   "syncai",
		Short: "Convert Cursor IDE rules to other AI tool formats",
		Long: `SyncAI converts Cursor IDE custom instructions and rules to formats
compatible with other AI development tools like WindSurf, Roo Code, and Cline.

It reads .cursorrules files and .cursor/rules directories, then generates
appropriate configuration files for each target tool.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if verbose {
				logger.SetLevel(logrus.DebugLevel)
			} else {
				logger.SetLevel(logrus.InfoLevel)
			}
		},
	}

	rootCmd.PersistentFlags().StringVarP(&outputDir, "output", "o", ".", "Output directory for generated files")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

	rootCmd.AddCommand(buildCmd())
	rootCmd.AddCommand(importCmd())

	if err := rootCmd.Execute(); err != nil {
		logger.Fatal(err)
	}
}

func buildCmd() *cobra.Command {
	var (
		targets    []string
		watchMode  bool
		projectDir string
	)

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build target AI tool configurations from Cursor rules",
		Long: `Parse .cursorrules and .cursor/rules files and convert them to
formats compatible with specified AI tools.

Supported targets: cursor, windsurf, roo-code, cline

Examples:
  syncai build --target roo-code
  syncai build --target windsurf --target cline
  syncai build --watch --target roo-code`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(targets) == 0 {
				return fmt.Errorf("at least one target must be specified")
			}

			// Validate and convert targets
			var targetTools []types.TargetTool
			for _, target := range targets {
				targetTool := types.TargetTool(target)
				if !targetTool.IsValid() {
					return fmt.Errorf("unsupported target: %s", target)
				}
				targetTools = append(targetTools, targetTool)
			}

			// Get project directory
			if projectDir == "" {
				var err error
				projectDir, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}
			}

			// Create parser and converter
			parser := config.NewParser(projectDir)
			converter := tools.NewConverter(outputDir, logger)

			// Initial build
			if err := performBuild(parser, converter, targetTools); err != nil {
				return err
			}

			// Watch mode
			if watchMode {
				return runWatchMode(projectDir, parser, converter, targetTools)
			}

			return nil
		},
	}

	cmd.Flags().StringSliceVarP(&targets, "target", "t", nil, "Target AI tools (cursor, windsurf, roo-code, cline)")
	cmd.Flags().BoolVarP(&watchMode, "watch", "w", false, "Watch for changes and rebuild automatically")
	cmd.Flags().StringVarP(&projectDir, "project", "p", "", "Project directory (default: current directory)")

	cmd.MarkFlagRequired("target")

	return cmd
}

func importCmd() *cobra.Command {
	var projectDir string

	cmd := &cobra.Command{
		Use:   "import [source-project]",
		Short: "Import rules from another project",
		Long: `Import .cursorrules and .cursor/rules files from another project
into the current project.

This command copies all cursor rules from the source project and merges
them with existing rules in the current project.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sourceDir := args[0]

			// Get target directory
			if projectDir == "" {
				var err error
				projectDir, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}
			}

			return performImport(sourceDir, projectDir)
		},
	}

	cmd.Flags().StringVarP(&projectDir, "project", "p", "", "Target project directory (default: current directory)")

	return cmd
}

func performBuild(parser *config.Parser, converter *tools.Converter, targets []types.TargetTool) error {
	logger.Info("Parsing cursor rules...")

	rules, err := parser.ParseCursorRules()
	if err != nil {
		return fmt.Errorf("failed to parse cursor rules: %w", err)
	}

	logger.Infof("Found %d MDC rules, %d folder rules", len(rules.MDCRules), len(rules.FolderRules))
	if rules.GlobalRules != "" {
		logger.Info("Found global rules")
	}

	targetNames := make([]string, len(targets))
	for i, target := range targets {
		targetNames[i] = string(target)
	}
	logger.Infof("Converting to targets: %s", strings.Join(targetNames, ", "))

	if err := converter.ConvertRules(rules, targets); err != nil {
		return fmt.Errorf("failed to convert rules: %w", err)
	}

	logger.Infof("Build completed successfully. Output written to: %s", outputDir)
	return nil
}

func runWatchMode(projectDir string, parser *config.Parser, converter *tools.Converter, targets []types.TargetTool) error {
	watcher, err := tools.NewWatcher(projectDir, converter, parser, targets, logger)
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Received interrupt signal, shutting down...")
		cancel()
	}()

	logger.Info("Watch mode enabled. Press Ctrl+C to stop.")
	return watcher.Start(ctx)
}

func performImport(sourceDir, targetDir string) error {
	logger.Infof("Importing rules from %s to %s", sourceDir, targetDir)

	// Parse source rules
	sourceParser := config.NewParser(sourceDir)
	sourceRules, err := sourceParser.ParseCursorRules()
	if err != nil {
		return fmt.Errorf("failed to parse source rules: %w", err)
	}

	// Copy global rules
	if sourceRules.GlobalRules != "" {
		targetPath := filepath.Join(targetDir, ".cursorrules")
		if err := writeFileWithMerge(targetPath, sourceRules.GlobalRules); err != nil {
			return fmt.Errorf("failed to copy global rules: %w", err)
		}
		logger.Info("Copied global rules")
	}

	// Copy folder rules
	for folderPath, content := range sourceRules.FolderRules {
		targetRulesDir := filepath.Join(targetDir, folderPath, ".cursor", "rules")
		if err := os.MkdirAll(targetRulesDir, 0755); err != nil {
			return fmt.Errorf("failed to create rules directory: %w", err)
		}

		targetPath := filepath.Join(targetRulesDir, "rules")
		if err := writeFileWithMerge(targetPath, content); err != nil {
			return fmt.Errorf("failed to copy folder rules for %s: %w", folderPath, err)
		}
		logger.Infof("Copied folder rules for %s", folderPath)
	}

	// Copy MDC rules
	for _, mdcRule := range sourceRules.MDCRules {
		// Create relative path in target directory
		relPath, err := filepath.Rel(sourceDir, mdcRule.FilePath)
		if err != nil {
			// If we can't get relative path, use the filename in .cursor/rules
			relPath = filepath.Join(".cursor", "rules", filepath.Base(mdcRule.FilePath))
		}

		targetPath := filepath.Join(targetDir, relPath)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create MDC directory: %w", err)
		}

		content := buildMDCContent(mdcRule)
		if err := writeFileWithMerge(targetPath, content); err != nil {
			return fmt.Errorf("failed to copy MDC rule %s: %w", mdcRule.Name, err)
		}
		logger.Infof("Copied MDC rule: %s", mdcRule.Name)
	}

	logger.Info("Import completed successfully")
	return nil
}

func writeFileWithMerge(path, content string) error {
	// Check if file exists
	if _, err := os.Stat(path); err == nil {
		// File exists, ask user for confirmation
		logger.Warnf("File %s already exists. Appending content...", path)

		existing, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Append new content with separator
		mergedContent := string(existing) + "\n\n# Imported content\n\n" + content
		return os.WriteFile(path, []byte(mergedContent), 0644)
	}

	// File doesn't exist, create it
	return os.WriteFile(path, []byte(content), 0644)
}

func buildMDCContent(rule types.MDCRule) string {
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
