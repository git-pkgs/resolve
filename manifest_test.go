package resolve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteManifest_Dispatch(t *testing.T) {
	dir := t.TempDir()
	deps := []InputDep{{Name: "x", Version: "1.0"}}

	supported := []string{"pip", "maven", "sbt", "pub", "mix", "lein"}
	for _, mgr := range supported {
		d := filepath.Join(dir, mgr)
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := writeManifest(d, mgr, deps); err != nil {
			t.Errorf("writeManifest(%q) returned error: %v", mgr, err)
		}
	}

	if err := writeManifest(dir, "nonexistent", deps); err == nil {
		t.Error("expected error for unknown manager")
	}
}

func TestWritePubManifest(t *testing.T) {
	dir := t.TempDir()
	deps := []InputDep{
		{Name: "http", Version: "^0.13.0"},
		{Name: "path", Version: ""},
	}

	if err := writePubManifest(dir, deps); err != nil {
		t.Fatalf("writePubManifest: %v", err)
	}

	got := readFile(t, filepath.Join(dir, "pubspec.yaml"))

	mustContain(t, got, "name: resolve_tmp")
	mustContain(t, got, "environment:")
	mustContain(t, got, "sdk:")
	mustContain(t, got, "dependencies:")
	mustContain(t, got, `  http: "^0.13.0"`)
	mustContain(t, got, "  path: any")
}

func TestWritePubManifest_NoDeps(t *testing.T) {
	dir := t.TempDir()
	if err := writePubManifest(dir, nil); err != nil {
		t.Fatalf("writePubManifest: %v", err)
	}
	got := readFile(t, filepath.Join(dir, "pubspec.yaml"))
	mustContain(t, got, "name: resolve_tmp")
	mustContain(t, got, "dependencies: {}")
}

func TestWriteMixManifest(t *testing.T) {
	dir := t.TempDir()
	deps := []InputDep{
		{Name: "phoenix", Version: "~> 1.7"},
		{Name: "jason", Version: ""},
	}

	if err := writeMixManifest(dir, deps); err != nil {
		t.Fatalf("writeMixManifest: %v", err)
	}

	got := readFile(t, filepath.Join(dir, "mix.exs"))

	mustContain(t, got, "defmodule ResolveTmp.MixProject do")
	mustContain(t, got, "use Mix.Project")
	mustContain(t, got, "app: :resolve_tmp")
	mustContain(t, got, `{:phoenix, "~> 1.7"}`)
	mustContain(t, got, `{:jason, ">= 0.0.0"}`)
}

func TestWriteMixManifest_NoDeps(t *testing.T) {
	dir := t.TempDir()
	if err := writeMixManifest(dir, nil); err != nil {
		t.Fatalf("writeMixManifest: %v", err)
	}
	got := readFile(t, filepath.Join(dir, "mix.exs"))
	mustContain(t, got, "defp deps do")
	mustContain(t, got, "[]")
}

func TestWriteLeinManifest(t *testing.T) {
	dir := t.TempDir()
	deps := []InputDep{
		{Name: "ring/ring-core", Version: "1.9.0"},
		{Name: "compojure", Version: ""},
	}

	if err := writeLeinManifest(dir, deps); err != nil {
		t.Fatalf("writeLeinManifest: %v", err)
	}

	got := readFile(t, filepath.Join(dir, "project.clj"))

	mustContain(t, got, "(defproject resolve-tmp")
	mustContain(t, got, ":dependencies [")
	mustContain(t, got, `[ring/ring-core "1.9.0"]`)
	mustContain(t, got, `[compojure "RELEASE"]`)
	mustContain(t, got, "[org.clojure/clojure")
}

func TestWriteLeinManifest_NoDeps(t *testing.T) {
	dir := t.TempDir()
	if err := writeLeinManifest(dir, nil); err != nil {
		t.Fatalf("writeLeinManifest: %v", err)
	}
	got := readFile(t, filepath.Join(dir, "project.clj"))
	// Even with no input deps, clojure itself is required
	mustContain(t, got, "[org.clojure/clojure")
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(b)
}

func mustContain(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected output to contain %q\n--- got ---\n%s", needle, haystack)
	}
}
