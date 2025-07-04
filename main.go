package main

import (
	"fmt"
	"os"

	"github.com/dudykr/syncai/internal/tools"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "syncai",
		Short: "Synchronize custom instructions across different AI tools",
		Long:  `A CLI tool to convert and synchronize custom instructions between different AI tools like Cursor, WindSurf, Roo Code, Cline, and Claude Code.`,
	}

	var buildCmd = &cobra.Command{
		Use:   "build",
		Short: "Build AI tool configuration files",
		Long:  `Build configuration files for specified AI tools from .cursorrules and .cursor/rules/*.mdc files.`,
		RunE:  runBuild,
	}

	var importCmd = &cobra.Command{
		Use:   "import",
		Short: "Import existing AI tool configurations",
		Long:  `Import existing AI tool configurations and convert them to the standard format.`,
		RunE:  runImport,
	}

	var targets []string
	var watch bool

	buildCmd.Flags().StringSliceVarP(&targets, "target", "t", []string{}, "Target AI tools (cursor, windsurf, roo-code, cline, claude-code)")
	buildCmd.Flags().BoolVarP(&watch, "watch", "w", false, "Watch for changes and rebuild automatically")

	rootCmd.AddCommand(buildCmd, importCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runBuild(cmd *cobra.Command, args []string) error {
	targets, _ := cmd.Flags().GetStringSlice("target")
	watch, _ := cmd.Flags().GetBool("watch")

	if len(targets) == 0 {
		targets = []string{"cursor", "windsurf", "roo-code", "cline", "claude-code"}
	}

	return tools.Build(targets, watch)
}

func runImport(cmd *cobra.Command, args []string) error {
	return tools.Import()
}
