package parsers

import (
	"regexp"
	"strings"

	"github.com/git-pkgs/resolve"
)

// composerPkgRe matches "vendor/package version" or "vendor/package version description".
var composerPkgRe = regexp.MustCompile(`^(\S+/\S+)\s+(\S+)`)

// parseComposer parses output from `composer show --tree`.
// Top-level packages are on unindented lines without tree markers.
// Their dependencies use box-drawing characters (├── └──).
func parseComposer(data []byte) ([]*resolve.Dep, error) {
	lines := strings.Split(string(data), "\n")
	opts := resolve.BoxDrawingOptions()

	var roots []*resolve.Dep

	type stackEntry struct {
		dep   *resolve.Dep
		depth int
	}
	var stack []stackEntry

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Check if line has tree markers (it's a sub-dependency)
		hasTreeMarker := false
		for _, prefix := range opts.Prefixes {
			if strings.Contains(line, prefix) {
				hasTreeMarker = true
				break
			}
		}
		for _, cont := range opts.Continuations {
			if strings.HasPrefix(line, cont) {
				hasTreeMarker = true
				break
			}
		}

		if !hasTreeMarker {
			// Top-level package line
			m := composerPkgRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			dep := &resolve.Dep{
				PURL:    resolve.MakePURL("packagist", m[1], m[2]),
				Name:    m[1],
				Version: m[2],
				Deps:    []*resolve.Dep{},
			}
			roots = append(roots, dep)
			stack = []stackEntry{{dep: dep, depth: -1}}
			continue
		}

		if len(stack) == 0 {
			continue
		}

		// Parse tree line for depth and content
		treeLines := resolve.ParseTreeLines([]string{line}, opts)
		if len(treeLines) == 0 {
			continue
		}

		tl := treeLines[0]
		m := composerPkgRe.FindStringSubmatch(tl.Content)
		if m == nil {
			continue
		}

		dep := &resolve.Dep{
			PURL:    resolve.MakePURL("packagist", m[1], m[2]),
			Name:    m[1],
			Version: m[2],
			Deps:    []*resolve.Dep{},
		}

		// Pop stack entries at same depth or deeper
		for len(stack) > 1 && stack[len(stack)-1].depth >= tl.Depth {
			stack = stack[:len(stack)-1]
		}

		parent := stack[len(stack)-1].dep
		parent.Deps = append(parent.Deps, dep)
		stack = append(stack, stackEntry{dep: dep, depth: tl.Depth})
	}

	return roots, nil
}

func init() {
	resolve.Register("composer", "packagist", parseComposer)
}
