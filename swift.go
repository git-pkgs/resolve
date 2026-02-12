package resolve

import (
	"encoding/json"
	"fmt"
)

// swiftPackage represents a package in swift's JSON output.
type swiftPackage struct {
	Name         string         `json:"name"`
	Version      string         `json:"version"`
	Dependencies []swiftPackage `json:"dependencies"`
}

// parseSwift parses output from `swift package show-dependencies --format json`.
func parseSwift(data []byte) ([]*Dep, error) {
	var root swiftPackage
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parsing swift output: %w", err)
	}
	// The root is the project itself; return its dependencies
	return walkSwiftDeps(root.Dependencies), nil
}

func walkSwiftDeps(pkgs []swiftPackage) []*Dep {
	var result []*Dep
	for _, pkg := range pkgs {
		dep := &Dep{
			PURL:    makePURL("swift", pkg.Name, pkg.Version),
			Name:    pkg.Name,
			Version: pkg.Version,
			Deps:    []*Dep{},
		}
		if len(pkg.Dependencies) > 0 {
			dep.Deps = walkSwiftDeps(pkg.Dependencies)
		}
		result = append(result, dep)
	}
	return result
}
