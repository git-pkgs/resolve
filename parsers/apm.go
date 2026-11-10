package parsers

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/git-pkgs/manifests"
	"go.yaml.in/yaml/v3"

	"github.com/git-pkgs/resolve"
)

// Errors reported for APM lockfile graphs.
var (
	ErrAPMLockfileMissing = errors.New("no apm.lock.yaml or apm.lock")
	ErrAPMAmbiguousParent = errors.New("ambiguous parent")
	ErrAPMMissingParent   = errors.New("parent not found")
	ErrAPMDuplicateEntry  = errors.New("duplicate lockfile entry")
)

type apmLock struct {
	Dependencies []apmLockEntry `yaml:"dependencies"`
}

type apmLockEntry struct {
	RepoURL        string `yaml:"repo_url"`
	Host           string `yaml:"host"`
	Source         string `yaml:"source"`
	LocalPath      string `yaml:"local_path"`
	IsVirtual      bool   `yaml:"is_virtual"`
	VirtualPath    string `yaml:"virtual_path"`
	RegistryPrefix string `yaml:"registry_prefix"`
	Depth          *int   `yaml:"depth"`
	ResolvedBy     string `yaml:"resolved_by"`
}

// apmNode is one lockfile entry with its identity and graph position.
type apmNode struct {
	entry apmLockEntry
	depth int
	key   string
	dep   *resolve.Dep
}

// parseAPMDir builds the graph from manifests' APM identities plus depth and resolved_by.
func parseAPMDir(dir string) ([]*resolve.Dep, error) {
	name, content, err := readAPMLockfile(dir)
	if err != nil {
		return nil, err
	}

	parsed, err := manifests.Parse(name, content)
	if err != nil {
		return nil, err
	}
	var lock apmLock
	if err := yaml.Unmarshal(content, &lock); err != nil {
		return nil, fmt.Errorf("reading %s: %w", name, err)
	}

	nodes := apmNodes(lock.Dependencies)
	if len(nodes) != len(parsed.Dependencies) {
		return nil, fmt.Errorf("reading %s: %d graph entries but %d dependencies", name, len(nodes), len(parsed.Dependencies))
	}
	keys := make(map[string]bool, len(nodes))
	for i, node := range nodes {
		if keys[node.key] {
			return nil, fmt.Errorf("%w: %s", ErrAPMDuplicateEntry, node.key)
		}
		keys[node.key] = true
		identity := parsed.Dependencies[i]
		node.dep = &resolve.Dep{PURL: identity.PURL, Name: identity.Name, Version: identity.Version, Deps: []*resolve.Dep{}}
	}

	var roots []*resolve.Dep
	for _, node := range nodes {
		if node.depth == 1 {
			roots = append(roots, node.dep)
			continue
		}
		parent, err := apmParent(nodes, node)
		if err != nil {
			return nil, err
		}
		parent.dep.Deps = append(parent.dep.Deps, node.dep)
	}
	return roots, nil
}

func readAPMLockfile(dir string) (string, []byte, error) {
	for _, name := range []string{"apm.lock.yaml", "apm.lock"} {
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err == nil {
			return name, content, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", nil, err
		}
	}
	return "", nil, ErrAPMLockfileMissing
}

// apmNodes keeps the entries manifests reports, skipping <self> and depth 0 as it does.
func apmNodes(entries []apmLockEntry) []*apmNode {
	var nodes []*apmNode
	for _, entry := range entries {
		depth := 1
		if entry.Depth != nil {
			depth = *entry.Depth
		}
		if entry.RepoURL == "<self>" || depth == 0 {
			continue
		}
		nodes = append(nodes, &apmNode{entry: entry, depth: depth, key: apmLockKey(entry)})
	}
	return nodes
}

// apmParent finds the single previous-depth entry named by resolved_by, as APM does.
func apmParent(nodes []*apmNode, child *apmNode) (*apmNode, error) {
	ref := child.entry.ResolvedBy
	for _, match := range []func(*apmNode) bool{
		func(n *apmNode) bool { return n.key == ref },
		func(n *apmNode) bool { return n.entry.RepoURL == ref },
	} {
		var candidates []*apmNode
		for _, n := range nodes {
			if n.depth == child.depth-1 && match(n) {
				candidates = append(candidates, n)
			}
		}
		switch len(candidates) {
		case 0:
			continue
		case 1:
			return candidates[0], nil
		default:
			return nil, fmt.Errorf("%w %q for %s: %d entries at depth %d", ErrAPMAmbiguousParent, ref, child.key, len(candidates), child.depth-1)
		}
	}
	return nil, fmt.Errorf("%w: %s at depth %d has resolved_by %q", ErrAPMMissingParent, child.key, child.depth, ref)
}

// apmLockKey mirrors APM's lockfile identity key.
func apmLockKey(entry apmLockEntry) string {
	if entry.Source == "local" && entry.LocalPath != "" {
		return entry.LocalPath
	}
	host := strings.ToLower(strings.TrimSpace(entry.Host))
	key := entry.RepoURL
	if entry.Source == "registry" || entry.RegistryPrefix != "" || host == "" || host == "github.com" || strings.HasSuffix(host, ".ghe.com") {
		key = strings.ToLower(key)
	}
	if entry.IsVirtual && entry.VirtualPath != "" {
		key += "/" + entry.VirtualPath
	}
	if entry.RegistryPrefix == "" && host != "" && host != "github.com" {
		key = host + "/" + key
	}
	return key
}

func init() {
	resolve.RegisterDir("apm", "apm", parseAPMDir)
}
