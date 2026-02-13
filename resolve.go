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

var managerEcosystem = map[string]string{}
var parsers = map[string]func([]byte) ([]*Dep, error){}

// Register adds a parser for a manager. Called from parser init() functions.
func Register(manager, ecosystem string, fn func([]byte) ([]*Dep, error)) {
	managerEcosystem[manager] = ecosystem
	parsers[manager] = fn
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

// MakePURL constructs a PURL string for a dependency.
func MakePURL(ecosystem, name, version string) string {
	return purl.MakePURL(ecosystem, name, version).String()
}
