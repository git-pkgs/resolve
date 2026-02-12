package resolve

import (
	"regexp"
	"strings"
)

// pubPkgRe matches "name version" in pub deps output.
var pubPkgRe = regexp.MustCompile(`^(\S+)\s+(\S+)`)

// parsePub parses output from `dart pub deps`.
// Box-drawing tree with ├── and └── markers. Packages formatted as "name version".
func parsePub(data []byte) ([]*Dep, error) {
	lines := strings.Split(string(data), "\n")

	// Skip header lines (everything before the first tree marker or package line)
	var treeStart int
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.Contains(line, "├──") || strings.Contains(line, "└──") ||
			strings.Contains(line, "|--") {
			treeStart = i
			break
		}
		// Lines that look like "package_name version" with no prefix
		if pubPkgRe.MatchString(trimmed) && !strings.Contains(trimmed, ":") {
			treeStart = i
			break
		}
	}

	opts := BoxDrawingOptions()
	treeLines := ParseTreeLines(lines[treeStart:], opts)

	return buildTree(treeLines, "pub", func(content string) (string, string, bool) {
		m := pubPkgRe.FindStringSubmatch(content)
		if m == nil {
			return "", "", false
		}
		return m[1], m[2], true
	}), nil
}
