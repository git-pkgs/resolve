package resolve

import (
	"errors"
	"fmt"

	"github.com/git-pkgs/purl"
)

// ErrUnsupportedManager is returned when Parse is called with an unknown manager name.
var ErrUnsupportedManager = errors.New("unsupported manager")

// Dep is a single resolved dependency.
type Dep struct {
	PURL    string // pkg:npm/%40scope/name@1.0.0
	Name    string // ecosystem-native name (@scope/name)
	Version string // resolved version (1.0.0)
	Deps    []*Dep // transitive deps; nil for flat-list managers
}

// Result is the parsed dependency graph for one manager invocation.
type Result struct {
	Manager   string // "npm", "cargo", etc.
	Ecosystem string // "npm", "cargo", "golang", etc.
	Direct    []*Dep // top-level dependencies
}

// managerEcosystem maps manager names to their ecosystem.
var managerEcosystem = map[string]string{
	"npm":      "npm",
	"pnpm":     "npm",
	"yarn":     "npm",
	"bun":      "npm",
	"cargo":    "cargo",
	"gomod":    "golang",
	"pip":      "pypi",
	"uv":       "pypi",
	"poetry":   "pypi",
	"conda":    "conda",
	"bundler":  "gem",
	"maven":    "maven",
	"gradle":   "maven",
	"composer": "packagist",
	"nuget":    "nuget",
	"swift":    "swift",
	"pub":      "pub",
	"mix":      "hex",
	"rebar3":   "hex",
	"stack":    "hackage",
	"lein":     "clojars",
	"conan":    "conan",
	"deno":     "deno",
	"helm":     "helm",
}

// parsers maps manager names to their parse function.
var parsers = map[string]func([]byte) ([]*Dep, error){
	"npm":      parseNPM,
	"pnpm":     parsePNPM,
	"yarn":     parseYarn,
	"bun":      parseBun,
	"cargo":    parseCargo,
	"gomod":    parseGomod,
	"pip":      parsePip,
	"uv":       parseUV,
	"poetry":   parsePoetry,
	"conda":    parseConda,
	"bundler":  parseBundler,
	"maven":    parseMaven,
	"gradle":   parseGradle,
	"composer": parseComposer,
	"nuget":    parseNuget,
	"swift":    parseSwift,
	"pub":      parsePub,
	"mix":      parseMix,
	"rebar3":   parseRebar3,
	"stack":    parseStack,
	"lein":     parseLein,
	"conan":    parseConan,
	"deno":     parseDeno,
	"helm":     parseHelm,
}

// Parse dispatches to the per-manager parser and returns the dependency graph.
func Parse(manager string, output []byte) (*Result, error) {
	eco, ok := managerEcosystem[manager]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedManager, manager)
	}

	parse, ok := parsers[manager]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedManager, manager)
	}

	deps, err := parse(output)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", manager, err)
	}

	return &Result{
		Manager:   manager,
		Ecosystem: eco,
		Direct:    deps,
	}, nil
}

// makePURL constructs a PURL string for a dependency.
func makePURL(ecosystem, name, version string) string {
	return purl.MakePURL(ecosystem, name, version).String()
}
