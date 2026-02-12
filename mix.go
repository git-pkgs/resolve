package resolve

import (
	"regexp"
	"strings"
)

// mixPkgRe matches "name version" or "name ~> constraint (Hex package)" patterns.
var mixPkgRe = regexp.MustCompile(`^(\S+)\s+(\S+)`)

// parseMix parses output from `mix deps.tree`.
func parseMix(data []byte) ([]*Dep, error) {
	lines := strings.Split(string(data), "\n")
	opts := BoxDrawingOptions()
	treeLines := ParseTreeLines(lines, opts)

	return buildTree(treeLines, "hex", func(content string) (string, string, bool) {
		m := mixPkgRe.FindStringSubmatch(content)
		if m == nil {
			return "", "", false
		}
		name := m[1]
		version := m[2]
		// Skip constraint-only entries (starting with ~>, >=, etc.)
		if strings.HasPrefix(version, "~>") || strings.HasPrefix(version, ">=") ||
			strings.HasPrefix(version, ">") || strings.HasPrefix(version, "<=") {
			return "", "", false
		}
		// Remove trailing parenthetical like "(Hex package)"
		return name, version, true
	}), nil
}
