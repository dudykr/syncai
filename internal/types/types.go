package types

import "time"

// TargetTool represents supported AI tools
type TargetTool string

const (
	TargetCursor   TargetTool = "cursor"
	TargetWindSurf TargetTool = "windsurf"
	TargetRooCode  TargetTool = "roo-code"
	TargetCline    TargetTool = "cline"
)

// IsValid checks if the target tool is supported
func (t TargetTool) IsValid() bool {
	switch t {
	case TargetCursor, TargetWindSurf, TargetRooCode, TargetCline:
		return true
	default:
		return false
	}
}

// CursorRules represents the main cursor rules configuration
type CursorRules struct {
	GlobalRules string            `yaml:"globalRules" json:"globalRules"`
	FolderRules map[string]string `yaml:"folderRules" json:"folderRules"`
	MDCRules    []MDCRule         `yaml:"mdcRules" json:"mdcRules"`
}

// MDCRule represents .cursor/rules/*.mdc file configuration
type MDCRule struct {
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description" json:"description"`
	Content     string   `yaml:"content" json:"content"`
	AlwaysApply bool     `yaml:"alwaysApply" json:"alwaysApply"`
	Globs       []string `yaml:"globs" json:"globs"`
	FilePath    string   `yaml:"filePath" json:"filePath"` // Original file path
}

// BuildConfig represents build configuration
type BuildConfig struct {
	Targets   []TargetTool `yaml:"targets" json:"targets"`
	OutputDir string       `yaml:"outputDir" json:"outputDir"`
	Watch     bool         `yaml:"watch" json:"watch"`
}

// ToolConfig represents configuration for a specific AI tool
type ToolConfig struct {
	Tool                TargetTool
	SupportsGlobalRules bool
	SupportsFolderRules bool
	SupportsMDCRules    bool
	FileExtension       string
	ConfigPath          string
	FolderConfigName    string
}

// GetToolConfigs returns configurations for all supported tools
func GetToolConfigs() map[TargetTool]ToolConfig {
	return map[TargetTool]ToolConfig{
		TargetCursor: {
			Tool:                TargetCursor,
			SupportsGlobalRules: true,
			SupportsFolderRules: true,
			SupportsMDCRules:    true,
			FileExtension:       "",
			ConfigPath:          ".cursorrules",
			FolderConfigName:    ".cursor/rules",
		},
		TargetWindSurf: {
			Tool:                TargetWindSurf,
			SupportsGlobalRules: true,
			SupportsFolderRules: false,
			SupportsMDCRules:    false,
			FileExtension:       "",
			ConfigPath:          ".windsurfrules",
			FolderConfigName:    "",
		},
		TargetRooCode: {
			Tool:                TargetRooCode,
			SupportsGlobalRules: true,
			SupportsFolderRules: true,
			SupportsMDCRules:    false,
			FileExtension:       ".md",
			ConfigPath:          "roo-code-rules.md",
			FolderConfigName:    ".roo-rules.md",
		},
		TargetCline: {
			Tool:                TargetCline,
			SupportsGlobalRules: true,
			SupportsFolderRules: false,
			SupportsMDCRules:    false,
			FileExtension:       ".md",
			ConfigPath:          ".cline/instructions.md",
			FolderConfigName:    "",
		},
	}
}

// WatchEvent represents a file system change event
type WatchEvent struct {
	Type      string    `json:"type"`
	Path      string    `json:"path"`
	Timestamp time.Time `json:"timestamp"`
}
