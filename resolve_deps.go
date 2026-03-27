package resolve

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/git-pkgs/managers"
	"github.com/git-pkgs/managers/definitions"
)

// InputDep describes a dependency to add to the generated project.
type InputDep struct {
	Name    string // package name in ecosystem-native format
	Version string // version constraint (optional)
}

// ErrResolveNotSupported is returned when the manager does not support the resolve operation.
var ErrResolveNotSupported = errors.New("manager does not support resolve")

// Managers returns the list of manager names that have registered parsers.
func Managers() []string {
	names := make([]string, 0, len(parsers))
	for name := range parsers {
		names = append(names, name)
	}
	return names
}

// EcosystemForManager returns the ecosystem name for a registered manager.
func EcosystemForManager(manager string) (string, bool) {
	eco, ok := managerEcosystem[manager]
	return eco, ok
}

// ResolveDeps creates a temporary project, adds the given dependencies using the
// specified package manager, runs resolution, and parses the output into a
// dependency graph.
//
// The package manager CLI must be installed and available on PATH.
func ResolveDeps(ctx context.Context, manager string, deps []InputDep) (*Result, error) {
	// Verify we have a parser for this manager.
	if _, ok := parsers[manager]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedManager, manager)
	}

	// Clear environment variables that might leak from parent processes
	// (e.g. BUNDLE_GEMFILE from a Rails app calling this binary).
	clearParentEnv()

	// Create temp directory for the project.
	tmpDir, err := os.MkdirTemp("", "resolve-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set up the managers library.
	defs, err := definitions.LoadEmbedded()
	if err != nil {
		return nil, fmt.Errorf("loading manager definitions: %w", err)
	}

	translator := managers.NewTranslator()
	runner := managers.NewExecRunner()
	detector := managers.NewDetector(translator, runner)
	for _, def := range defs {
		detector.Register(def)
	}

	mgr, err := detector.Detect(tmpDir, managers.DetectOptions{Manager: manager})
	if err != nil {
		return nil, fmt.Errorf("setting up manager %s: %w", manager, err)
	}

	// Init the project and add dependencies.
	if mgr.Supports(managers.CapInit) {
		result, err := mgr.Init(ctx)
		if err != nil {
			return nil, fmt.Errorf("init %s: %w", manager, err)
		}
		if !result.Success() {
			return nil, fmt.Errorf("init %s: exit %d: %s", manager, result.ExitCode, result.Stderr)
		}

		// Re-detect after init (some commands create subdirectories).
		mgr, err = detector.Detect(tmpDir, managers.DetectOptions{Manager: manager})
		if err != nil {
			return nil, fmt.Errorf("re-detecting manager after init: %w", err)
		}

		// Add each dependency via the manager CLI.
		if mgr.Supports(managers.CapAdd) {
			seen := make(map[string]bool)
			for _, dep := range deps {
				if seen[dep.Name] {
					continue
				}
				seen[dep.Name] = true

				result, err := mgr.Add(ctx, dep.Name, managers.AddOptions{Version: dep.Version})
				if err != nil {
					return nil, fmt.Errorf("add %s: %w", dep.Name, err)
				}
				if !result.Success() {
					return nil, fmt.Errorf("add %s: exit %d: %s", dep.Name, result.ExitCode, result.Stderr)
				}
			}
		}
	} else {
		// Fallback: write a minimal manifest for managers without init.
		if err := writeManifest(tmpDir, manager, deps); err != nil {
			return nil, fmt.Errorf("writing manifest for %s: %w", manager, err)
		}

		// Re-detect so the manager sees the manifest.
		mgr, err = detector.Detect(tmpDir, managers.DetectOptions{Manager: manager})
		if err != nil {
			return nil, fmt.Errorf("detecting manager after manifest write: %w", err)
		}

		// Run install to resolve dependencies.
		installResult, err := mgr.Install(ctx, managers.InstallOptions{})
		if err != nil {
			return nil, fmt.Errorf("install %s: %w", manager, err)
		}
		if !installResult.Success() {
			return nil, fmt.Errorf("install %s: exit %d: %s", manager, installResult.ExitCode, installResult.Stderr)
		}
	}

	// Run resolve to get the dependency graph output.
	if !mgr.Supports(managers.CapResolve) {
		return nil, fmt.Errorf("%w: %s", ErrResolveNotSupported, manager)
	}

	resolveResult, err := mgr.Resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", manager, err)
	}

	// Parse the output.
	return Parse(manager, []byte(resolveResult.Stdout))
}

// writeManifest creates a minimal manifest file for managers that don't support init.
func writeManifest(dir, manager string, deps []InputDep) error {
	switch manager {
	case "pip":
		return writePipManifest(dir, deps)
	case "maven":
		return writeMavenManifest(dir, deps)
	case "sbt":
		return writeSbtManifest(dir, deps)
	default:
		return fmt.Errorf("no manifest template for manager %s", manager)
	}
}

func writePipManifest(dir string, deps []InputDep) error {
	var lines []string
	for _, dep := range deps {
		if dep.Version != "" {
			lines = append(lines, dep.Name+dep.Version)
		} else {
			lines = append(lines, dep.Name)
		}
	}
	return os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

func writeMavenManifest(dir string, deps []InputDep) error {
	var depXML strings.Builder
	for _, dep := range deps {
		// Maven deps use groupId:artifactId format
		parts := strings.SplitN(dep.Name, ":", 2)
		groupID := parts[0]
		artifactID := groupID
		if len(parts) == 2 {
			artifactID = parts[1]
		}
		version := dep.Version
		if version == "" {
			version = "[0,)"
		}
		depXML.WriteString("    <dependency>\n")
		depXML.WriteString("      <groupId>" + groupID + "</groupId>\n")
		depXML.WriteString("      <artifactId>" + artifactID + "</artifactId>\n")
		depXML.WriteString("      <version>" + version + "</version>\n")
		depXML.WriteString("    </dependency>\n")
	}

	pom := `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
  <modelVersion>4.0.0</modelVersion>
  <groupId>resolve</groupId>
  <artifactId>resolve-tmp</artifactId>
  <version>0.0.1</version>
  <dependencies>
` + depXML.String() + `  </dependencies>
</project>
`
	return os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pom), 0644)
}

func writeSbtManifest(dir string, deps []InputDep) error {
	var lines []string
	lines = append(lines, `name := "resolve-tmp"`)
	lines = append(lines, `version := "0.0.1"`)
	lines = append(lines, "")

	var depLines []string
	for _, dep := range deps {
		parts := strings.SplitN(dep.Name, ":", 2)
		groupID := parts[0]
		artifactID := groupID
		if len(parts) == 2 {
			artifactID = parts[1]
		}
		version := dep.Version
		if version == "" {
			version = "LATEST"
		}
		depLines = append(depLines, fmt.Sprintf(`  "%s" %% "%s" %% "%s"`, groupID, artifactID, version))
	}
	if len(depLines) > 0 {
		lines = append(lines, "libraryDependencies ++= Seq(")
		lines = append(lines, strings.Join(depLines, ",\n"))
		lines = append(lines, ")")
	}

	return os.WriteFile(filepath.Join(dir, "build.sbt"), []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

// envVarsToClear lists specific environment variables that point to a parent
// project and would cause package manager commands to operate on the wrong
// directory (e.g. BUNDLE_GEMFILE from a Rails app).
var envVarsToClear = []string{
	"BUNDLE_GEMFILE",
	"BUNDLE_LOCKFILE",
	"BUNDLE_BIN_PATH",
	"BUNDLE_PATH",
	"BUNDLER_SETUP",
	"BUNDLER_VERSION",
	"GEM_HOME",
	"GEM_PATH",
	"RUBYOPT",
	"RUBYLIB",
}

// clearParentEnv removes environment variables that could interfere with
// package manager commands run in a temporary directory.
func clearParentEnv() {
	for _, key := range envVarsToClear {
		os.Unsetenv(key)
	}
}
