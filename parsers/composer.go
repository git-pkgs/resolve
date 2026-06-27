package parsers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/git-pkgs/resolve"
)

type composerManifest struct {
	Require map[string]string `json:"require"`
}

type composerLock struct {
	Packages    []composerLockPackage `json:"packages"`
	PackagesDev []composerLockPackage `json:"packages-dev"`
}

type composerLockPackage struct {
	Name    string            `json:"name"`
	Version string            `json:"version"`
	Require map[string]string `json:"require"`
}

// parseComposerDir reads composer.json and composer.lock from dir and builds
// the dependency graph. composer.lock provides resolved versions for every
// installed package plus each package's require map; composer.json identifies
// the direct dependencies that become tree roots.
func parseComposerDir(dir string) ([]*resolve.Dep, error) {
	var manifest composerManifest
	if err := readJSON(filepath.Join(dir, "composer.json"), &manifest); err != nil {
		return nil, fmt.Errorf("reading composer.json: %w", err)
	}

	var lock composerLock
	if err := readJSON(filepath.Join(dir, "composer.lock"), &lock); err != nil {
		return nil, fmt.Errorf("reading composer.lock: %w", err)
	}

	packages := make(map[string]composerLockPackage)
	for _, pkg := range lock.Packages {
		packages[pkg.Name] = pkg
	}
	for _, pkg := range lock.PackagesDev {
		packages[pkg.Name] = pkg
	}

	seen := make(map[string]bool)
	var build func(name string) *resolve.Dep
	build = func(name string) *resolve.Dep {
		pkg, ok := packages[name]
		if !ok {
			return nil
		}
		dep := &resolve.Dep{
			PURL:    resolve.MakePURL("packagist", pkg.Name, normaliseComposerVersion(pkg.Version)),
			Name:    pkg.Name,
			Version: normaliseComposerVersion(pkg.Version),
			Deps:    []*resolve.Dep{},
		}
		if seen[name] {
			return dep
		}
		seen[name] = true
		for childName := range pkg.Require {
			if isComposerPlatformPackage(childName) {
				continue
			}
			if child := build(childName); child != nil {
				dep.Deps = append(dep.Deps, child)
			}
		}
		return dep
	}

	var roots []*resolve.Dep
	for name := range manifest.Require {
		if isComposerPlatformPackage(name) {
			continue
		}
		if dep := build(name); dep != nil {
			roots = append(roots, dep)
		}
	}
	return roots, nil
}

// isComposerPlatformPackage reports whether name is a Composer platform
// requirement (php, hhvm, ext-*, lib-*, composer runtime/plugin APIs) rather
// than an installable Packagist package.
func isComposerPlatformPackage(name string) bool {
	if name == "php" || name == "hhvm" {
		return true
	}
	if strings.HasPrefix(name, "php-") {
		return true
	}
	if strings.HasPrefix(name, "ext-") || strings.HasPrefix(name, "lib-") {
		return true
	}
	if name == "composer" || strings.HasPrefix(name, "composer-") {
		return true
	}
	return false
}

// normaliseComposerVersion strips a leading "v" so versions match Packagist's
// canonical form (composer.lock stores tags like "v7.1.0").
func normaliseComposerVersion(v string) string {
	if len(v) > 1 && (v[0] == 'v' || v[0] == 'V') && v[1] >= '0' && v[1] <= '9' {
		return v[1:]
	}
	return v
}

func readJSON(path string, dst any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

func init() {
	resolve.RegisterDir("composer", "packagist", parseComposerDir)
}
