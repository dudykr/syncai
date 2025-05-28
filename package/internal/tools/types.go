package tools

// A markdown file that contains instructions for the tool.
type MdcFile struct {
	Path        string
	Description string
	Globs       []string
	AlwaysApply bool
	// Markdown content of the file
	Content string
}
