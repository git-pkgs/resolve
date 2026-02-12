package resolve

import (
	"encoding/json"
	"fmt"
)

// parsePip parses output from `pip inspect`.
// Format: {"installed": [{"metadata": {"name": "...", "version": "..."}}, ...]}
func parsePip(data []byte) ([]*Dep, error) {
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

	var deps []*Dep
	for _, pkg := range output.Installed {
		name := pkg.Metadata.Name
		version := pkg.Metadata.Version
		if name == "" {
			continue
		}
		deps = append(deps, &Dep{
			PURL:    makePURL("pypi", name, version),
			Name:    name,
			Version: version,
		})
	}
	return deps, nil
}

// parseConda parses output from `conda list --json`.
// Format: [{"name": "...", "version": "..."}, ...]
func parseConda(data []byte) ([]*Dep, error) {
	var packages []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &packages); err != nil {
		return nil, fmt.Errorf("parsing conda output: %w", err)
	}

	var deps []*Dep
	for _, pkg := range packages {
		if pkg.Name == "" {
			continue
		}
		deps = append(deps, &Dep{
			PURL:    makePURL("conda", pkg.Name, pkg.Version),
			Name:    pkg.Name,
			Version: pkg.Version,
		})
	}
	return deps, nil
}

// parseStack parses output from `stack ls dependencies json`.
// Format: [{"name": "...", "version": "..."}, ...]
func parseStack(data []byte) ([]*Dep, error) {
	var packages []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &packages); err != nil {
		return nil, fmt.Errorf("parsing stack output: %w", err)
	}

	var deps []*Dep
	for _, pkg := range packages {
		if pkg.Name == "" {
			continue
		}
		deps = append(deps, &Dep{
			PURL:    makePURL("hackage", pkg.Name, pkg.Version),
			Name:    pkg.Name,
			Version: pkg.Version,
		})
	}
	return deps, nil
}
