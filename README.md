# SyncAI

A CLI tool to synchronize custom instructions across different AI tools. Convert and sync your custom instructions between Cursor IDE, WindSurf, Roo Code, Cline, and Claude Code.

## Features

- **Universal Compatibility**: Supports 5 major AI development tools
- **Watch Mode**: Automatically rebuild configurations when source files change
- **Parallel Processing**: Build configurations for multiple tools simultaneously
- **MDC Support**: Full support for Cursor's `.mdc` rule files with `alwaysApply` and `globs`
- **Import/Export**: Import existing configurations from any supported tool

## Supported AI Tools

| Tool | Input Format | Output Format |
|------|-------------|---------------|
| **Cursor IDE** | `.cursorrules`, `.cursor/rules/*.mdc` | Native (no conversion needed) |
| **WindSurf** | `.cursorrules`, `.cursor/rules/*.mdc` | `.windsurfrules` |
| **Roo Code** | `.cursorrules`, `.cursor/rules/*.mdc` | `.roocode/*.md` |
| **Cline** | `.cursorrules`, `.cursor/rules/*.mdc` | `.clinerules` |
| **Claude Code** | `.cursorrules`, `.cursor/rules/*.mdc` | `CLAUDE.md` |

## Installation

```bash
go install github.com/dudykr/syncai@latest
```

## Usage

### Build Configurations

Generate configuration files for specific AI tools:

```bash
# Build for specific tools
syncai build --target cursor --target windsurf

# Build for all supported tools
syncai build

# Build with watch mode (auto-rebuild on file changes)
syncai build --watch
```

### Import Existing Configurations

Detect and import existing AI tool configurations:

```bash
syncai import
```

### Available Targets

- `cursor` - Cursor IDE (validates existing files)
- `windsurf` - WindSurf (generates `.windsurfrules`)
- `roo-code` - Roo Code (generates `.roocode/*.md`)
- `cline` - Cline (generates `.clinerules`)
- `claude-code` - Claude Code (generates `CLAUDE.md`)

## Configuration Files

### Global Rules (`.cursorrules`)

The `.cursorrules` file contains global instructions that apply to the entire project:

```markdown
# Global Development Rules

## Code Style
- Use TypeScript for all new JavaScript files
- Follow ESLint configuration strictly
- Use meaningful variable names

## Testing
- Write unit tests for all new functions
- Use Jest for testing framework
- Aim for 80% test coverage
```

### Context-Specific Rules (`.cursor/rules/*.mdc`)

MDC files provide context-specific instructions with file pattern matching:

```markdown
---
description: React Component Rules
globs: ["**/*.tsx", "**/*.jsx"]
alwaysApply: false
---

# React Component Guidelines

## Component Structure
- Use functional components with hooks
- Keep components small and focused
- Use TypeScript interfaces for props
```

#### MDC File Structure

- **Frontmatter**: YAML metadata between `---` lines
  - `description`: Human-readable description of the rules
  - `globs`: Array of file patterns where rules apply
  - `alwaysApply`: Boolean indicating if rules should always be active
- **Content**: Markdown content with the actual instructions

## Project Structure

```
your-project/
├── .cursorrules                 # Global rules
├── .cursor/
│   └── rules/
│       ├── react.mdc           # React-specific rules
│       ├── api.mdc             # API-specific rules
│       └── ...
├── src/
│   └── components/
│       └── .cursor/
│           └── rules/
│               └── styling.mdc  # Component-specific rules
└── ...
```

## How It Works

1. **Discovery**: SyncAI scans your project for:
   - `.cursorrules` file in the project root
   - All `.cursor` directories (can be nested anywhere)
   - All `.mdc` files in `.cursor/rules/` directories

2. **Parsing**: Parses MDC files to extract:
   - Metadata (description, globs, alwaysApply)
   - Markdown content with instructions

3. **Transformation**: Converts rules to each target tool's format:
   - **WindSurf**: Combines all rules into `.windsurfrules`
   - **Roo Code**: Creates separate `.md` files in `.roocode/`
   - **Cline**: Generates `.clinerules` file
   - **Claude Code**: Generates comprehensive `CLAUDE.md`

4. **Parallel Processing**: Builds configurations for all specified tools simultaneously

## Examples

### Basic Usage

```bash
# Generate configurations for all tools
syncai build

# Generate for specific tools only
syncai build --target windsurf --target roo-code

# Watch for changes and auto-rebuild
syncai build --watch
```

### Testing

Test the build functionality:

```bash
cd examples/build-test
syncai build --target windsurf --target roo-code --target cline --target claude-code
```

Test the import functionality:

```bash
cd examples/import-test
syncai import
```

## Watch Mode

When running with `--watch`, SyncAI monitors:
- `.cursorrules` file changes
- All `.cursor/rules/` directories
- Creation/modification of `.mdc` files

Changes trigger automatic rebuilds with a 100ms debounce to handle rapid file changes.

## Error Handling

- **Missing Files**: Gracefully handles missing configuration files
- **Invalid MDC**: Logs warnings for unparseable MDC files but continues processing
- **Permission Errors**: Reports file permission issues clearly
- **Parallel Processing**: Individual tool failures don't stop other tools from building

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for new functionality
5. Commit your changes (`git commit -m 'Add some amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Roadmap

- [ ] Support for more AI tools
- [ ] Configuration validation
- [ ] Template system for common rule sets
- [ ] GUI interface
- [ ] Rule inheritance and composition
- [ ] Integration with popular IDEs

## Support

- Create an issue for bug reports or feature requests
- Check the [examples/](examples/) directory for usage examples
- Review the source code for implementation details