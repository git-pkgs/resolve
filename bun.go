package resolve

import (
	"strings"
)

// parseBun parses output from `bun pm ls --all`.
// Tree output with box-drawing, similar to npm text output.
// Format: name@version
func parseBun(data []byte) ([]*Dep, error) {
	lines := strings.Split(string(data), "\n")

	// Skip the root line (first non-empty line is the project)
	startIdx := 0
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		// First tree marker indicates start of deps
		if strings.Contains(line, "├──") || strings.Contains(line, "└──") {
			startIdx = i
			break
		}
	}

	opts := BoxDrawingOptions()
	treeLines := ParseTreeLines(lines[startIdx:], opts)

	return buildTree(treeLines, "npm", func(content string) (string, string, bool) {
		// Format: name@version or @scope/name@version
		name, version := parseAtVersion(content)
		if name == "" {
			return "", "", false
		}
		return name, version, true
	}), nil
}

// parseAtVersion splits "name@version" or "@scope/name@version".
func parseAtVersion(s string) (string, string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	// For scoped packages (@scope/name@version), last @ is the version separator
	idx := strings.LastIndex(s, "@")
	if idx <= 0 {
		return s, ""
	}
	return s[:idx], s[idx+1:]
}
