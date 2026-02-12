package resolve

import (
	"regexp"
	"strings"
)

// leinPkgRe matches "[group/name \"version\"]" or "[name \"version\"]".
var leinPkgRe = regexp.MustCompile(`\[(\S+)\s+"([^"]+)"\]`)

// parseLein parses output from `lein deps :tree`.
// Bracket-indented format: [group/name "version"] with increasing space indentation.
func parseLein(data []byte) ([]*Dep, error) {
	lines := strings.Split(string(data), "\n")
	var treeLines []TreeLine

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		m := leinPkgRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		// Calculate depth from leading whitespace
		depth := 0
		for _, ch := range line {
			if ch == ' ' {
				depth++
			} else {
				break
			}
		}
		// Normalize depth: each level is typically 2 spaces
		depth = depth / 2

		name := m[1]
		version := m[2]

		treeLines = append(treeLines, TreeLine{Depth: depth, Content: name + "\t" + version})
	}

	return buildTree(treeLines, "clojars", func(content string) (string, string, bool) {
		parts := strings.SplitN(content, "\t", 2)
		if len(parts) != 2 {
			return "", "", false
		}
		return parts[0], parts[1], true
	}), nil
}
