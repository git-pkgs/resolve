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

const manifestPerm = 0o644

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
	if _, ok := parsers[manager]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedManager, manager)
	}

	clearParentEnv()

	tmpDir, err := os.MkdirTemp("", "resolve-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	detector, err := newDetector()
	if err != nil {
		return nil, err
	}

	mgr, err := detector.Detect(tmpDir, managers.DetectOptions{Manager: manager})
	if err != nil {
		return nil, fmt.Errorf("setting up manager %s: %w", manager, err)
	}

	if mgr.Supports(managers.CapInit) {
		mgr, err = initAndAdd(ctx, mgr, detector, tmpDir, manager, deps)
	} else {
		mgr, err = writeAndInstall(ctx, mgr, detector, tmpDir, manager, deps)
	}
	if err != nil {
		return nil, err
	}

	if !mgr.Supports(managers.CapResolve) {
		return nil, fmt.Errorf("%w: %s", ErrResolveNotSupported, manager)
	}

	resolveResult, err := mgr.Resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", manager, err)
	}

	return Parse(manager, []byte(resolveResult.Stdout))
}

func newDetector() (*managers.Detector, error) {
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
	return detector, nil
}

func initAndAdd(ctx context.Context, mgr managers.Manager, detector *managers.Detector, tmpDir, manager string, deps []InputDep) (managers.Manager, error) { //nolint:ireturn
	result, err := mgr.Init(ctx)
	if err != nil {
		return nil, fmt.Errorf("init %s: %w", manager, err)
	}
	if !result.Success() {
		return nil, fmt.Errorf("init %s: exit %d: %s", manager, result.ExitCode, result.Stderr)
	}

	mgr, err = detector.Detect(tmpDir, managers.DetectOptions{Manager: manager})
	if err != nil {
		return nil, fmt.Errorf("re-detecting manager after init: %w", err)
	}

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

	return mgr, nil
}

func writeAndInstall(ctx context.Context, _ managers.Manager, detector *managers.Detector, tmpDir, manager string, deps []InputDep) (managers.Manager, error) { //nolint:ireturn
	if err := writeManifest(tmpDir, manager, deps); err != nil {
		return nil, fmt.Errorf("writing manifest for %s: %w", manager, err)
	}

	mgr, err := detector.Detect(tmpDir, managers.DetectOptions{Manager: manager})
	if err != nil {
		return nil, fmt.Errorf("detecting manager after manifest write: %w", err)
	}

	installResult, err := mgr.Install(ctx, managers.InstallOptions{})
	if err != nil {
		return nil, fmt.Errorf("install %s: %w", manager, err)
	}
	if !installResult.Success() {
		return nil, fmt.Errorf("install %s: exit %d: %s", manager, installResult.ExitCode, installResult.Stderr)
	}

	return mgr, nil
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
	case "pub":
		return writePubManifest(dir, deps)
	case "mix":
		return writeMixManifest(dir, deps)
	case "lein":
		return writeLeinManifest(dir, deps)
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
	return os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(strings.Join(lines, "\n")+"\n"), manifestPerm)
}

func writeMavenManifest(dir string, deps []InputDep) error {
	var depXML strings.Builder
	for _, dep := range deps {
		groupID, artifactID := splitGroupArtifact(dep.Name)
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
	return os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pom), manifestPerm)
}

func writeSbtManifest(dir string, deps []InputDep) error {
	var lines []string
	lines = append(lines, `name := "resolve-tmp"`)
	lines = append(lines, `version := "0.0.1"`)
	lines = append(lines, "")

	var depLines []string
	for _, dep := range deps {
		groupID, artifactID := splitGroupArtifact(dep.Name)
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

	return os.WriteFile(filepath.Join(dir, "build.sbt"), []byte(strings.Join(lines, "\n")+"\n"), manifestPerm)
}

func writePubManifest(dir string, deps []InputDep) error {
	var b strings.Builder
	b.WriteString("name: resolve_tmp\n")
	b.WriteString("environment:\n")
	b.WriteString("  sdk: '>=2.12.0 <4.0.0'\n")

	if len(deps) == 0 {
		b.WriteString("dependencies: {}\n")
	} else {
		b.WriteString("dependencies:\n")
		for _, dep := range deps {
			if dep.Version == "" {
				fmt.Fprintf(&b, "  %s: any\n", dep.Name)
			} else {
				fmt.Fprintf(&b, "  %s: %q\n", dep.Name, dep.Version)
			}
		}
	}

	return os.WriteFile(filepath.Join(dir, "pubspec.yaml"), []byte(b.String()), manifestPerm)
}

func writeMixManifest(dir string, deps []InputDep) error {
	var depLines []string
	for _, dep := range deps {
		version := dep.Version
		if version == "" {
			version = ">= 0.0.0"
		}
		depLines = append(depLines, fmt.Sprintf(`      {:%s, %q}`, dep.Name, version))
	}
	depList := "[]"
	if len(depLines) > 0 {
		depList = "[\n" + strings.Join(depLines, ",\n") + "\n    ]"
	}

	src := `defmodule ResolveTmp.MixProject do
  use Mix.Project

  def project do
    [
      app: :resolve_tmp,
      version: "0.1.0",
      elixir: "~> 1.12",
      deps: deps()
    ]
  end

  def application do
    [extra_applications: [:logger]]
  end

  defp deps do
    ` + depList + `
  end
end
`
	return os.WriteFile(filepath.Join(dir, "mix.exs"), []byte(src), manifestPerm)
}

func writeLeinManifest(dir string, deps []InputDep) error {
	// org.clojure/clojure must always be present or lein deps fails.
	depLines := []string{`[org.clojure/clojure "1.11.1"]`}
	for _, dep := range deps {
		version := dep.Version
		if version == "" {
			version = "RELEASE"
		}
		depLines = append(depLines, fmt.Sprintf(`[%s %q]`, dep.Name, version))
	}

	src := `(defproject resolve-tmp "0.1.0"
  :dependencies [` + strings.Join(depLines, "\n                 ") + `])
`
	return os.WriteFile(filepath.Join(dir, "project.clj"), []byte(src), manifestPerm)
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
		_ = os.Unsetenv(key)
	}
}

func splitGroupArtifact(name string) (groupID, artifactID string) {
	groupID, artifactID, found := strings.Cut(name, ":")
	if !found {
		artifactID = groupID
	}
	return groupID, artifactID
}
