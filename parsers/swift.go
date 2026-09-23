package parsers

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/git-pkgs/resolve"
)

// swiftPackage represents a package in swift's JSON output.
type swiftPackage struct {
	Name         string         `json:"name"`
	URL          string         `json:"url"`
	Version      string         `json:"version"`
	Dependencies []swiftPackage `json:"dependencies"`
}

// parseSwift parses output from `swift package show-dependencies --format json`.
func parseSwift(data []byte) ([]*resolve.Dep, error) {
	var root swiftPackage
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parsing swift output: %w", err)
	}
	// The root is the project itself; return its dependencies
	return walkSwiftDeps(root.Dependencies), nil
}

func walkSwiftDeps(pkgs []swiftPackage) []*resolve.Dep {
	var result []*resolve.Dep
	for _, pkg := range pkgs {
		dep := &resolve.Dep{
			PURL:    swiftPURL(pkg),
			Name:    pkg.Name,
			Version: pkg.Version,
			Deps:    []*resolve.Dep{},
		}
		if len(pkg.Dependencies) > 0 {
			dep.Deps = walkSwiftDeps(pkg.Dependencies)
		}
		result = append(result, dep)
	}
	return result
}

func swiftPURL(pkg swiftPackage) string {
	source := pkg.URL
	if strings.HasPrefix(source, "git@") {
		source = "ssh://" + strings.Replace(source, ":", "/", 1)
	}
	u, err := url.Parse(source)
	if err != nil || u.Hostname() == "" || u.Scheme == "file" {
		return ""
	}
	name := u.Hostname() + strings.TrimSuffix(u.Path, ".git")
	return resolve.MakePURL("swift", name, pkg.Version)
}

func init() {
	resolve.Register("swift", "swift", parseSwift)
}
