package resolve_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/git-pkgs/resolve"
	"github.com/git-pkgs/resolve/parsers"
)

const (
	apmSkillsCommit = "7f3c9a4d2e1b8c7f0a9e6d5c4b3a2918f7e6d5c4"
	apmCommit       = "0123456789abcdef0123456789abcdef01234567"
)

func writeAPMLockfile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestAPM(t *testing.T) {
	result, err := resolve.ParseDir("apm", filepath.Join("testdata", "apm"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Manager != "apm" || result.Ecosystem != "apm" {
		t.Errorf("result = %s/%s, want apm/apm", result.Manager, result.Ecosystem)
	}

	type node struct{ name, version, purl string }
	var roots []node
	for _, dep := range result.Direct {
		roots = append(roots, node{dep.Name, dep.Version, dep.PURL})
	}
	want := []node{
		{"github.com/acme-corp/security-baseline", "a1b2c3d4e5f6789012345678901234567890abcd",
			"pkg:git/github.com/acme-corp/security-baseline@a1b2c3d4e5f6789012345678901234567890abcd"},
		{"github.com/composiohq/awesome-claude-skills/brand-guidelines", apmSkillsCommit,
			"pkg:git/github.com/composiohq/awesome-claude-skills@" + apmSkillsCommit + "#brand-guidelines"},
		{"github.com/composiohq/awesome-claude-skills/canvas-design", apmSkillsCommit,
			"pkg:git/github.com/composiohq/awesome-claude-skills@" + apmSkillsCommit + "#canvas-design"},
		// Registry and local sources have no package coordinates.
		{"acme/toolkit", "2.3.1", ""},
		{"./packages/shared-skills", "0.1.0", ""},
		{"github.com/owner/test-helpers", apmCommit, "pkg:git/github.com/owner/test-helpers@" + apmCommit},
	}
	if !reflect.DeepEqual(roots, want) {
		t.Fatalf("direct =\n%+v\nwant\n%+v", roots, want)
	}

	// Transitive chain resolved by repo_url, across hosts.
	baseline := findDep(result.Direct, "github.com/acme-corp/security-baseline")
	prompts := findDep(baseline.Deps, "github.com/acme-corp/common-prompts")
	if prompts == nil || len(baseline.Deps) != 1 {
		t.Fatalf("security-baseline deps = %+v", baseline.Deps)
	}
	style := findDep(prompts.Deps, "gitlab.com/team/style-guide")
	if style == nil || style.PURL != "pkg:git/gitlab.com/team/style-guide@0f1e2d3c4b5a69788796a5b4c3d2e1f0a9b8c7d6" {
		t.Errorf("style-guide = %+v", style)
	}

	// A parent named by its full lock key picks one subpackage, not its sibling.
	canvas := findDep(result.Direct, "github.com/composiohq/awesome-claude-skills/canvas-design")
	if findDep(canvas.Deps, "github.com/composiohq/design-tokens") == nil {
		t.Errorf("canvas-design deps = %+v, want design-tokens", canvas.Deps)
	}
	brand := findDep(result.Direct, "github.com/composiohq/awesome-claude-skills/brand-guidelines")
	if len(brand.Deps) != 0 {
		t.Errorf("brand-guidelines deps = %+v, want none", brand.Deps)
	}

	for _, dep := range result.Direct {
		if dep.Deps == nil {
			t.Errorf("%s Deps is nil, want an empty slice for a tree manager", dep.Name)
		}
	}
}

func TestAPMLegacyLockfile(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("testdata", "apm", "apm.lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	current, err := resolve.ParseDir("apm", filepath.Join("testdata", "apm"))
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := resolve.ParseDir("apm", writeAPMLockfile(t, "apm.lock", string(content)))
	if err != nil {
		t.Fatalf("legacy apm.lock: %v", err)
	}
	if !reflect.DeepEqual(legacy.Direct, current.Direct) {
		t.Error("legacy apm.lock graph differs from apm.lock.yaml")
	}
}

func TestAPMPrefersCurrentLockfile(t *testing.T) {
	dir := writeAPMLockfile(t, "apm.lock.yaml", "dependencies:\n  - repo_url: owner/current\n    resolved_commit: "+apmCommit+"\n")
	if err := os.WriteFile(filepath.Join(dir, "apm.lock"), []byte("dependencies:\n  - repo_url: owner/legacy\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := resolve.ParseDir("apm", dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Direct) != 1 || result.Direct[0].Name != "github.com/owner/current" {
		t.Errorf("direct = %+v, want only github.com/owner/current", result.Direct)
	}
}

func TestAPMParentByHostQualifiedKey(t *testing.T) {
	result, err := resolve.ParseDir("apm", writeAPMLockfile(t, "apm.lock.yaml", `dependencies:
  - repo_url: team/rules
    host: gitlab.example.com
    resolved_commit: `+apmCommit+`
  - repo_url: team/shared
    host: gitlab.example.com
    resolved_commit: `+apmCommit+`
    depth: 2
    resolved_by: gitlab.example.com/team/rules
`))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Direct) != 1 || findDep(result.Direct[0].Deps, "gitlab.example.com/team/shared") == nil {
		t.Errorf("direct = %+v, want team/shared under team/rules", result.Direct)
	}
}

func TestAPMGraphErrors(t *testing.T) {
	commit := "    resolved_commit: " + apmCommit + "\n"
	tests := []struct {
		name, content string
		want          error
	}{
		{"ambiguous parent", "dependencies:\n" +
			"  - repo_url: acme/skills\n    virtual_path: review\n    is_virtual: true\n" + commit +
			"  - repo_url: acme/skills\n    virtual_path: deploy\n    is_virtual: true\n" + commit +
			"  - repo_url: acme/shared\n" + commit + "    depth: 2\n    resolved_by: acme/skills\n",
			parsers.ErrAPMAmbiguousParent},
		{"missing parent", "dependencies:\n" +
			"  - repo_url: acme/orphan\n" + commit + "    depth: 2\n    resolved_by: acme/gone\n",
			parsers.ErrAPMMissingParent},
		{"parent at wrong depth", "dependencies:\n" +
			"  - repo_url: acme/root\n" + commit +
			"  - repo_url: acme/deep\n" + commit + "    depth: 3\n    resolved_by: acme/root\n",
			parsers.ErrAPMMissingParent},
		{"cyclic parents", "dependencies:\n" +
			"  - repo_url: acme/a\n" + commit + "    depth: 2\n    resolved_by: acme/b\n" +
			"  - repo_url: acme/b\n" + commit + "    depth: 2\n    resolved_by: acme/a\n",
			parsers.ErrAPMMissingParent},
		{"duplicate entry", "dependencies:\n" +
			"  - repo_url: acme/twice\n" + commit +
			"  - repo_url: acme/twice\n" + commit,
			parsers.ErrAPMDuplicateEntry},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := resolve.ParseDir("apm", writeAPMLockfile(t, "apm.lock.yaml", tt.content))
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestAPMInvalidLockfile(t *testing.T) {
	for name, content := range map[string]string{
		"malformed YAML":      "dependencies: [unterminated\n",
		"missing repo_url":    "dependencies:\n  - resolved_commit: " + apmCommit + "\n",
		"unsupported version": "lockfile_version: \"3\"\nreplacements: []\ndependencies: []\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := resolve.ParseDir("apm", writeAPMLockfile(t, "apm.lock.yaml", content)); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestAPMMissingLockfile(t *testing.T) {
	_, err := resolve.ParseDir("apm", t.TempDir())
	if !errors.Is(err, parsers.ErrAPMLockfileMissing) {
		t.Errorf("err = %v, want ErrAPMLockfileMissing", err)
	}
}
