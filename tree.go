package resolve

import (
	"strings"
)

// TreeLine is a parsed line from a text tree.
type TreeLine struct {
	Depth   int
	Content string
}

// TreeOptions configures how tree lines are parsed.
type TreeOptions struct {
	// Prefixes are the tree-drawing characters that indicate a child node.
	// e.g. "├── ", "└── ", "+- ", "\- "
	Prefixes []string

	// Continuations are the tree-drawing characters that indicate depth continuation.
	// e.g. "│   ", "|  ", "│  "
	Continuations []string
}

// BoxDrawingOptions returns TreeOptions for Unicode box-drawing trees (├── └── │).
func BoxDrawingOptions() TreeOptions {
	return TreeOptions{
		Prefixes:      []string{"├── ", "└── "},
		Continuations: []string{"│   ", "    "},
	}
}

// ParseTreeLines reads indented tree output and returns depth + content for each line.
func ParseTreeLines(lines []string, opts TreeOptions) []TreeLine {
	var result []TreeLine
	for _, line := range lines {
		if line == "" {
			continue
		}
		depth := 0
		remaining := line
		for {
			found := false
			for _, cont := range opts.Continuations {
				if strings.HasPrefix(remaining, cont) {
					depth++
					remaining = remaining[len(cont):]
					found = true
					break
				}
			}
			if !found {
				break
			}
		}
		for _, prefix := range opts.Prefixes {
			if strings.HasPrefix(remaining, prefix) {
				remaining = remaining[len(prefix):]
				break
			}
		}
		content := strings.TrimRight(remaining, " \t\r\n")
		if content == "" {
			continue
		}
		result = append(result, TreeLine{Depth: depth, Content: content})
	}
	return result
}

// BuildTree takes tree lines and a content-parser function, and builds a []*Dep tree.
// The contentParser receives the content string and returns (name, version, deps-placeholder).
// Deps is set to non-nil empty slice to indicate tree structure is available.
func BuildTree(lines []TreeLine, ecosystem string, contentParser func(string) (string, string, bool)) []*Dep {
	if len(lines) == 0 {
		return nil
	}

	type stackEntry struct {
		dep   *Dep
		depth int
	}

	var roots []*Dep
	var stack []stackEntry

	for _, line := range lines {
		name, version, ok := contentParser(line.Content)
		if !ok {
			continue
		}

		dep := &Dep{
			PURL:    MakePURL(ecosystem, name, version),
			Name:    name,
			Version: version,
			Deps:    []*Dep{}, // non-nil to indicate tree structure
		}

		// Pop stack entries that are at the same depth or deeper
		for len(stack) > 0 && stack[len(stack)-1].depth >= line.Depth {
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			roots = append(roots, dep)
		} else {
			parent := stack[len(stack)-1].dep
			parent.Deps = append(parent.Deps, dep)
		}

		stack = append(stack, stackEntry{dep: dep, depth: line.Depth})
	}

	return roots
}
