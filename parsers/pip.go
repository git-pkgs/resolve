package parsers

import (
	"encoding/json"
	"fmt"

	"github.com/git-pkgs/resolve"
)

// parsePip parses output from `pip inspect`.
// Format: {"installed": [{"metadata": {"name": "...", "version": "..."}}, ...]}
func parsePip(data []byte) ([]*resolve.Dep, error) {
	var output struct {
		Installed []struct {
			Metadata struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"metadata"`
		} `json:"installed"`
	}
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("parsing pip output: %w", err)
	}

	var deps []*resolve.Dep
	for _, pkg := range output.Installed {
		name := pkg.Metadata.Name
		version := pkg.Metadata.Version
		if name == "" {
			continue
		}
		deps = append(deps, &resolve.Dep{
			PURL:    resolve.MakePURL("pypi", name, version),
			Name:    name,
			Version: version,
		})
	}
	return deps, nil
}

// parseConda parses output from `conda list --json`.
// Format: [{"name": "...", "version": "..."}, ...]
func parseConda(data []byte) ([]*resolve.Dep, error) {
	var packages []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &packages); err != nil {
		return nil, fmt.Errorf("parsing conda output: %w", err)
	}

	var deps []*resolve.Dep
	for _, pkg := range packages {
		if pkg.Name == "" {
			continue
		}
		deps = append(deps, &resolve.Dep{
			PURL:    resolve.MakePURL("conda", pkg.Name, pkg.Version),
			Name:    pkg.Name,
			Version: pkg.Version,
		})
	}
	return deps, nil
}

// parseStack parses output from `stack ls dependencies json`.
// Format: [{"name": "...", "version": "..."}, ...]
func parseStack(data []byte) ([]*resolve.Dep, error) {
	var packages []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &packages); err != nil {
		return nil, fmt.Errorf("parsing stack output: %w", err)
	}

	var deps []*resolve.Dep
	for _, pkg := range packages {
		if pkg.Name == "" {
			continue
		}
		deps = append(deps, &resolve.Dep{
			PURL:    resolve.MakePURL("hackage", pkg.Name, pkg.Version),
			Name:    pkg.Name,
			Version: pkg.Version,
		})
	}
	return deps, nil
}

func init() {
	resolve.Register("pip", "pypi", parsePip)
	resolve.Register("conda", "conda", parseConda)
	resolve.Register("stack", "hackage", parseStack)
}
