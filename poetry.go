package resolve

import (
	"regexp"
	"strings"
)

// poetryTopRe matches top-level lines like "requests 2.31.0 Description here".
var poetryTopRe = regexp.MustCompile(`^(\S+)\s+(\S+)(?:\s+.*)?$`)

// poetrySubRe matches sub-dependency lines like "├── certifi (>=2017.4.17)".
var poetrySubRe = regexp.MustCompile(`^(\S+)\s+`)

// parsePoetry parses output from `poetry show --tree --no-ansi`.
// Top-level packages appear on unindented lines: "name version description".
// Sub-deps use box-drawing and show constraints, not resolved versions.
// We cross-reference sub-deps against top-level entries for actual versions.
func parsePoetry(data []byte) ([]*Dep, error) {
	lines := strings.Split(string(data), "\n")

	// First pass: collect all top-level package versions
	versions := make(map[string]string)
	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "│") ||
			strings.HasPrefix(line, "├") || strings.HasPrefix(line, "└") {
			continue
		}
		m := poetryTopRe.FindStringSubmatch(line)
		if m != nil {
			versions[strings.ToLower(m[1])] = m[2]
		}
	}

	// Second pass: build tree
	opts := BoxDrawingOptions()
	var result []*Dep
	var currentTop *Dep

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Top-level line (no indentation, no box-drawing prefix)
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "│") &&
			!strings.HasPrefix(line, "├") && !strings.HasPrefix(line, "└") {
			m := poetryTopRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			currentTop = &Dep{
				PURL:    makePURL("pypi", m[1], m[2]),
				Name:    m[1],
				Version: m[2],
				Deps:    []*Dep{},
			}
			result = append(result, currentTop)
			continue
		}

		if currentTop == nil {
			continue
		}

		// Sub-dependency line - parse with tree helper for depth
		treeLines := ParseTreeLines([]string{line}, opts)
		if len(treeLines) == 0 {
			continue
		}

		content := treeLines[0].Content
		m := poetrySubRe.FindStringSubmatch(content)
		if m == nil {
			continue
		}

		name := m[1]
		version := versions[strings.ToLower(name)]

		if treeLines[0].Depth == 0 {
			// Direct sub-dep of current top-level package
			currentTop.Deps = append(currentTop.Deps, &Dep{
				PURL:    makePURL("pypi", name, version),
				Name:    name,
				Version: version,
				Deps:    []*Dep{},
			})
		}
	}

	return result, nil
}
