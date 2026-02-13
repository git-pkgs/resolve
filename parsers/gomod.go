package parsers

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/git-pkgs/resolve"
)

// parseGomod parses output from `go mod graph`.
// Format: one edge per line, space-separated: "parent@version dep@version"
// Root module has no @version suffix.
func parseGomod(data []byte) ([]*resolve.Dep, error) {
	type edge struct {
		from, to string
	}

	var edges []edge
	modules := make(map[string]bool)
	children := make(map[string][]string)
	root := ""

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}

		from := parts[0]
		to := parts[1]
		edges = append(edges, edge{from, to})
		modules[from] = true
		modules[to] = true

		// Root is the module without @version
		if !strings.Contains(from, "@") && root == "" {
			root = from
		}

		children[from] = append(children[from], to)
	}

	if root == "" && len(edges) > 0 {
		root = edges[0].from
	}

	// Build tree from root's direct children
	seen := make(map[string]bool)
	var buildDeps func(mod string) *resolve.Dep
	buildDeps = func(mod string) *resolve.Dep {
		name, version := splitModVersion(mod)
		dep := &resolve.Dep{
			PURL:    resolve.MakePURL("golang", name, version),
			Name:    name,
			Version: version,
			Deps:    []*resolve.Dep{},
		}
		if seen[mod] {
			return dep
		}
		seen[mod] = true
		for _, child := range children[mod] {
			dep.Deps = append(dep.Deps, buildDeps(child))
		}
		return dep
	}

	var deps []*resolve.Dep
	for _, child := range children[root] {
		deps = append(deps, buildDeps(child))
	}
	return deps, nil
}

func splitModVersion(s string) (string, string) {
	if idx := strings.LastIndex(s, "@"); idx > 0 {
		return s[:idx], s[idx+1:]
	}
	return s, ""
}

func init() {
	resolve.Register("gomod", "golang", parseGomod)
}
