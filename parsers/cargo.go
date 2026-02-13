package parsers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/git-pkgs/resolve"
)

// parseCargo parses output from `cargo metadata --format-version 1`.
func parseCargo(data []byte) ([]*resolve.Dep, error) {
	var meta struct {
		Packages []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			ID      string `json:"id"`
		} `json:"packages"`
		Resolve struct {
			Root  string `json:"root"`
			Nodes []struct {
				ID   string `json:"id"`
				Deps []struct {
					Pkg string `json:"pkg"`
				} `json:"deps"`
			} `json:"nodes"`
		} `json:"resolve"`
	}

	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parsing cargo output: %w", err)
	}

	// Build lookup from package ID to name+version
	type pkgInfo struct {
		Name    string
		Version string
	}
	lookup := make(map[string]pkgInfo)
	for _, pkg := range meta.Packages {
		lookup[pkg.ID] = pkgInfo{Name: pkg.Name, Version: pkg.Version}
	}

	// Build adjacency list
	children := make(map[string][]string)
	for _, node := range meta.Resolve.Nodes {
		for _, dep := range node.Deps {
			children[node.ID] = append(children[node.ID], dep.Pkg)
		}
	}

	// Find root
	root := meta.Resolve.Root
	if root == "" && len(meta.Resolve.Nodes) > 0 {
		root = meta.Resolve.Nodes[0].ID
	}

	// Walk from root
	seen := make(map[string]bool)
	var buildDep func(id string) *resolve.Dep
	buildDep = func(id string) *resolve.Dep {
		info, ok := lookup[id]
		if !ok {
			// Try to extract from ID format: "name version (source)"
			info = parseCargoID(id)
		}
		dep := &resolve.Dep{
			PURL:    resolve.MakePURL("cargo", info.Name, info.Version),
			Name:    info.Name,
			Version: info.Version,
			Deps:    []*resolve.Dep{},
		}
		if seen[id] {
			return dep
		}
		seen[id] = true
		for _, child := range children[id] {
			dep.Deps = append(dep.Deps, buildDep(child))
		}
		return dep
	}

	var deps []*resolve.Dep
	for _, child := range children[root] {
		deps = append(deps, buildDep(child))
	}
	return deps, nil
}

func parseCargoID(id string) struct{ Name, Version string } {
	// Cargo IDs look like "name version (source)" or "name version"
	parts := strings.Fields(id)
	if len(parts) >= 2 {
		return struct{ Name, Version string }{parts[0], parts[1]}
	}
	return struct{ Name, Version string }{id, ""}
}

func init() {
	resolve.Register("cargo", "cargo", parseCargo)
}
