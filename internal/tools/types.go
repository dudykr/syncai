package tools

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// A markdown file that contains instructions for the tool.
type MdcFile struct {
	Path        string
	Description string
	Globs       []string
	AlwaysApply bool
	// Markdown content of the file
	Content string
}

// ProjectConfig represents the configuration for a project
type ProjectConfig struct {
	RootPath     string
	CursorRules  string
	MdcFiles     []MdcFile
	CursorDirs   []string
}

// AITool represents an AI tool configuration
type AITool interface {
	Name() string
	Build(config *ProjectConfig) error
	Import(rootPath string) (*ProjectConfig, error)
}

// Build builds configuration files for the specified AI tools
func Build(targets []string, watch bool) error {
	config, err := loadProjectConfig()
	if err != nil {
		return fmt.Errorf("failed to load project config: %w", err)
	}

	tools := make([]AITool, 0, len(targets))
	for _, target := range targets {
		tool, err := createTool(target)
		if err != nil {
			return fmt.Errorf("failed to create tool %s: %w", target, err)
		}
		tools = append(tools, tool)
	}

	if watch {
		return watchAndBuild(config, tools)
	}

	return buildOnce(config, tools)
}

// Import imports existing AI tool configurations
func Import() error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	fmt.Printf("Importing AI tool configurations from %s...\n", wd)

	// Check what AI tools are already configured
	tools := []string{"cursor", "windsurf", "roo-code", "cline", "claude-code"}
	found := []string{}
	
	for _, toolName := range tools {
		tool, err := createTool(toolName)
		if err != nil {
			continue
		}
		
		config, err := tool.Import(wd)
		if err != nil {
			continue
		}
		
		if config.CursorRules != "" || len(config.MdcFiles) > 0 {
			found = append(found, toolName)
		}
	}
	
	if len(found) == 0 {
		fmt.Printf("  ⚠ No AI tool configurations found to import\n")
		return nil
	}
	
	fmt.Printf("  ✓ Found configurations for: %s\n", strings.Join(found, ", "))
	
	// For now, we'll focus on importing from the first found tool
	// In a real implementation, you might want to ask the user which one to import from
	if len(found) > 0 {
		fmt.Printf("  → Use 'syncai build' to generate configurations for other tools\n")
	}
	
	return nil
}

func loadProjectConfig() (*ProjectConfig, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	config := &ProjectConfig{
		RootPath: wd,
	}

	// Load .cursorrules file
	cursorRulesPath := filepath.Join(wd, ".cursorrules")
	if data, err := os.ReadFile(cursorRulesPath); err == nil {
		config.CursorRules = string(data)
	}

	// Find all .cursor directories
	cursorDirs := []string{}
	err = filepath.Walk(wd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == ".cursor" {
			cursorDirs = append(cursorDirs, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find .cursor directories: %w", err)
	}

	config.CursorDirs = cursorDirs

	// Load MDC files from all .cursor/rules directories
	mdcFiles := []MdcFile{}
	for _, cursorDir := range cursorDirs {
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
					log.Printf("Warning: failed to parse MDC file %s: %v", path, err)
					return nil
				}
				mdcFiles = append(mdcFiles, *mdcFile)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to walk rules directory %s: %w", rulesDir, err)
		}
	}

	config.MdcFiles = mdcFiles

	return config, nil
}

func parseMdcFile(path string) (*MdcFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	mdcFile := &MdcFile{
		Path:    path,
		Content: content,
	}

	// Parse frontmatter-like metadata
	inFrontmatter := false
	contentStart := 0
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			} else {
				contentStart = i + 1
				break
			}
		}
		if inFrontmatter {
			if strings.HasPrefix(line, "description:") {
				mdcFile.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			} else if strings.HasPrefix(line, "alwaysApply:") {
				mdcFile.AlwaysApply = strings.TrimSpace(strings.TrimPrefix(line, "alwaysApply:")) == "true"
			} else if strings.HasPrefix(line, "globs:") {
				globsStr := strings.TrimSpace(strings.TrimPrefix(line, "globs:"))
				if strings.HasPrefix(globsStr, "[") && strings.HasSuffix(globsStr, "]") {
					globsStr = strings.Trim(globsStr, "[]")
					globs := strings.Split(globsStr, ",")
					for i, glob := range globs {
						globs[i] = strings.Trim(strings.TrimSpace(glob), "\"'")
					}
					mdcFile.Globs = globs
				}
			}
		}
	}

	if contentStart > 0 {
		mdcFile.Content = strings.Join(lines[contentStart:], "\n")
	}

	return mdcFile, nil
}

func createTool(name string) (AITool, error) {
	switch name {
	case "cursor":
		return &Cursor{}, nil
	case "windsurf":
		return &WindSurf{}, nil
	case "roo-code":
		return &RooCode{}, nil
	case "cline":
		return &Cline{}, nil
	case "claude-code":
		return &ClaudeCode{}, nil
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func buildOnce(config *ProjectConfig, tools []AITool) error {
	var wg sync.WaitGroup
	errors := make(chan error, len(tools))

	for _, tool := range tools {
		wg.Add(1)
		go func(t AITool) {
			defer wg.Done()
			if err := t.Build(config); err != nil {
				errors <- fmt.Errorf("failed to build %s: %w", t.Name(), err)
			}
		}(tool)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			return err
		}
	}

	return nil
}

func watchAndBuild(config *ProjectConfig, tools []AITool) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	defer watcher.Close()

	// Add files to watch
	cursorRulesPath := filepath.Join(config.RootPath, ".cursorrules")
	if _, err := os.Stat(cursorRulesPath); err == nil {
		err = watcher.Add(cursorRulesPath)
		if err != nil {
			return fmt.Errorf("failed to watch .cursorrules: %w", err)
		}
	}

	for _, cursorDir := range config.CursorDirs {
		rulesDir := filepath.Join(cursorDir, "rules")
		if _, err := os.Stat(rulesDir); err == nil {
			err = watcher.Add(rulesDir)
			if err != nil {
				return fmt.Errorf("failed to watch rules directory %s: %w", rulesDir, err)
			}
		}
	}

	// Initial build
	if err := buildOnce(config, tools); err != nil {
		return fmt.Errorf("initial build failed: %w", err)
	}

	fmt.Println("Watching for changes... Press Ctrl+C to stop.")

	// Watch for changes
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				fmt.Printf("File modified: %s\n", event.Name)
				
				// Debounce: wait a bit for multiple rapid changes
				time.Sleep(100 * time.Millisecond)
				
				// Reload config and rebuild
				newConfig, err := loadProjectConfig()
				if err != nil {
					log.Printf("Failed to reload config: %v", err)
					continue
				}
				
				if err := buildOnce(newConfig, tools); err != nil {
					log.Printf("Build failed: %v", err)
				} else {
					fmt.Println("Build completed successfully")
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}
