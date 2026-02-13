package parsers

import (
	"regexp"
	"strings"

	"github.com/git-pkgs/resolve"
)

// mixPkgRe matches "name version" or "name ~> constraint (Hex package)" patterns.
var mixPkgRe = regexp.MustCompile(`^(\S+)\s+(\S+)`)

// parseMix parses output from `mix deps.tree`.
func parseMix(data []byte) ([]*resolve.Dep, error) {
	lines := strings.Split(string(data), "\n")
	opts := resolve.BoxDrawingOptions()
	treeLines := resolve.ParseTreeLines(lines, opts)

	return resolve.BuildTree(treeLines, "hex", func(content string) (string, string, bool) {
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

func init() {
	resolve.Register("mix", "hex", parseMix)
}
