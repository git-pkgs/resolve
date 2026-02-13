package parsers

import (
	"encoding/json"
	"fmt"

	"github.com/git-pkgs/resolve"
)

// npmPackage represents a package in npm/pnpm JSON output.
type npmPackage struct {
	Version         string                `json:"version"`
	Dependencies    map[string]npmPackage `json:"dependencies"`
	DevDependencies map[string]npmPackage `json:"devDependencies"`
}

// parseNPM parses output from `npm ls --depth Infinity --json --long`.
func parseNPM(data []byte) ([]*resolve.Dep, error) {
	var root npmPackage
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parsing npm output: %w", err)
	}
	return walkNPMDeps(root.Dependencies, "npm"), nil
}

// parsePNPM parses output from `pnpm list --json --depth Infinity`.
// PNPM returns a JSON array of workspace entries.
func parsePNPM(data []byte) ([]*resolve.Dep, error) {
	var entries []npmPackage
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parsing pnpm output: %w", err)
	}
	var deps []*resolve.Dep
	for _, entry := range entries {
		deps = append(deps, walkNPMDeps(entry.Dependencies, "npm")...)
		deps = append(deps, walkNPMDeps(entry.DevDependencies, "npm")...)
	}
	return deps, nil
}

func walkNPMDeps(deps map[string]npmPackage, ecosystem string) []*resolve.Dep {
	var result []*resolve.Dep
	for name, pkg := range deps {
		dep := &resolve.Dep{
			PURL:    resolve.MakePURL(ecosystem, name, pkg.Version),
			Name:    name,
			Version: pkg.Version,
			Deps:    []*resolve.Dep{},
		}
		if len(pkg.Dependencies) > 0 {
			dep.Deps = walkNPMDeps(pkg.Dependencies, ecosystem)
		}
		result = append(result, dep)
	}
	return result
}

func init() {
	resolve.Register("npm", "npm", parseNPM)
	resolve.Register("pnpm", "npm", parsePNPM)
}
