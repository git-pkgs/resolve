package resolve_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/git-pkgs/resolve"
	_ "github.com/git-pkgs/resolve/parsers"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("loading fixture %s: %v", name, err)
	}
	return data
}

func findDep(deps []*resolve.Dep, name string) *resolve.Dep {
	for _, d := range deps {
		if d.Name == name {
			return d
		}
	}
	return nil
}

func TestParseUnsupportedManager(t *testing.T) {
	_, err := resolve.Parse("unknown-manager", []byte("{}"))
	if !errors.Is(err, resolve.ErrUnsupportedManager) {
		t.Errorf("expected ErrUnsupportedManager, got %v", err)
	}
}

func TestParseReturnsManagerAndEcosystem(t *testing.T) {
	result, err := resolve.Parse("pip", loadFixture(t, "pip.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Manager != "pip" {
		t.Errorf("Manager = %q, want %q", result.Manager, "pip")
	}
	if result.Ecosystem != "pypi" {
		t.Errorf("Ecosystem = %q, want %q", result.Ecosystem, "pypi")
	}
}

func TestNPM(t *testing.T) {
	result, err := resolve.Parse("npm", loadFixture(t, "npm.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Direct) != 2 {
		t.Fatalf("expected 2 direct deps, got %d", len(result.Direct))
	}

	express := findDep(result.Direct, "express")
	if express == nil {
		t.Fatal("missing express")
	}
	if express.Version != "4.18.2" {
		t.Errorf("express version = %q, want %q", express.Version, "4.18.2")
	}
	if !strings.Contains(express.PURL, "pkg:npm/express@4.18.2") {
		t.Errorf("express PURL = %q, want pkg:npm/express@4.18.2", express.PURL)
	}
	if len(express.Deps) != 2 {
		t.Errorf("express transitive deps = %d, want 2", len(express.Deps))
	}

	babel := findDep(result.Direct, "@babel/core")
	if babel == nil {
		t.Fatal("missing @babel/core")
	}
	if !strings.Contains(babel.PURL, "%40babel") {
		t.Errorf("babel PURL should contain encoded scope, got %q", babel.PURL)
	}

	// Check transitive dep
	accepts := findDep(express.Deps, "accepts")
	if accepts == nil {
		t.Fatal("missing accepts under express")
	}
	if len(accepts.Deps) != 1 {
		t.Errorf("accepts transitive deps = %d, want 1", len(accepts.Deps))
	}
}

func TestPNPM(t *testing.T) {
	result, err := resolve.Parse("pnpm", loadFixture(t, "pnpm.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Ecosystem != "npm" {
		t.Errorf("Ecosystem = %q, want %q", result.Ecosystem, "npm")
	}
	// 2 deps + 1 devDep = 3
	if len(result.Direct) != 3 {
		t.Fatalf("expected 3 direct deps, got %d", len(result.Direct))
	}
	axios := findDep(result.Direct, "axios")
	if axios == nil {
		t.Fatal("missing axios")
	}
	if len(axios.Deps) != 1 {
		t.Errorf("axios transitive deps = %d, want 1", len(axios.Deps))
	}
}

func TestYarn(t *testing.T) {
	result, err := resolve.Parse("yarn", loadFixture(t, "yarn.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Direct) != 3 {
		t.Fatalf("expected 3 direct deps, got %d", len(result.Direct))
	}
	react := findDep(result.Direct, "react")
	if react == nil {
		t.Fatal("missing react")
	}
	if react.Version != "18.2.0" {
		t.Errorf("react version = %q, want %q", react.Version, "18.2.0")
	}
	if len(react.Deps) != 1 {
		t.Errorf("react transitive deps = %d, want 1", len(react.Deps))
	}

	// Scoped package
	types := findDep(result.Direct, "@types/node")
	if types == nil {
		t.Fatal("missing @types/node")
	}
	if !strings.Contains(types.PURL, "%40types") {
		t.Errorf("@types/node PURL should have encoded scope, got %q", types.PURL)
	}
}

func TestBun(t *testing.T) {
	result, err := resolve.Parse("bun", loadFixture(t, "bun.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Direct) != 2 {
		t.Fatalf("expected 2 direct deps, got %d", len(result.Direct))
	}
	express := findDep(result.Direct, "express")
	if express == nil {
		t.Fatal("missing express")
	}
	if express.Version != "4.18.2" {
		t.Errorf("express version = %q, want %q", express.Version, "4.18.2")
	}
	if len(express.Deps) != 2 {
		t.Errorf("express transitive deps = %d, want 2", len(express.Deps))
	}
}

func TestCargo(t *testing.T) {
	result, err := resolve.Parse("cargo", loadFixture(t, "cargo.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Direct) != 2 {
		t.Fatalf("expected 2 direct deps, got %d", len(result.Direct))
	}
	serde := findDep(result.Direct, "serde")
	if serde == nil {
		t.Fatal("missing serde")
	}
	if serde.Version != "1.0.193" {
		t.Errorf("serde version = %q, want %q", serde.Version, "1.0.193")
	}
	if !strings.Contains(serde.PURL, "pkg:cargo/serde@1.0.193") {
		t.Errorf("serde PURL = %q", serde.PURL)
	}
	// serde depends on serde_derive
	if len(serde.Deps) != 1 {
		t.Errorf("serde transitive deps = %d, want 1", len(serde.Deps))
	}
}

func TestGomod(t *testing.T) {
	result, err := resolve.Parse("gomod", loadFixture(t, "gomod.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Ecosystem != "golang" {
		t.Errorf("Ecosystem = %q, want %q", result.Ecosystem, "golang")
	}
	if len(result.Direct) != 2 {
		t.Fatalf("expected 2 direct deps, got %d", len(result.Direct))
	}

	text := findDep(result.Direct, "golang.org/x/text")
	if text == nil {
		t.Fatal("missing golang.org/x/text")
	}
	if text.Version != "v0.14.0" {
		t.Errorf("text version = %q, want %q", text.Version, "v0.14.0")
	}
	// text depends on tools
	if len(text.Deps) != 1 {
		t.Errorf("text transitive deps = %d, want 1", len(text.Deps))
	}

	testify := findDep(result.Direct, "github.com/stretchr/testify")
	if testify == nil {
		t.Fatal("missing testify")
	}
	if len(testify.Deps) != 2 {
		t.Errorf("testify transitive deps = %d, want 2", len(testify.Deps))
	}
}

func TestPip(t *testing.T) {
	result, err := resolve.Parse("pip", loadFixture(t, "pip.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Direct) != 3 {
		t.Fatalf("expected 3 deps, got %d", len(result.Direct))
	}
	requests := findDep(result.Direct, "requests")
	if requests == nil {
		t.Fatal("missing requests")
	}
	// Flat list: Deps should be nil
	if requests.Deps != nil {
		t.Error("pip deps should have nil Deps (flat list)")
	}
	if !strings.Contains(requests.PURL, "pkg:pypi/requests@2.31.0") {
		t.Errorf("requests PURL = %q", requests.PURL)
	}
}

func TestConda(t *testing.T) {
	result, err := resolve.Parse("conda", loadFixture(t, "conda.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Direct) != 3 {
		t.Fatalf("expected 3 deps, got %d", len(result.Direct))
	}
	numpy := findDep(result.Direct, "numpy")
	if numpy == nil {
		t.Fatal("missing numpy")
	}
	if numpy.Deps != nil {
		t.Error("conda deps should have nil Deps (flat list)")
	}
}

func TestStack(t *testing.T) {
	result, err := resolve.Parse("stack", loadFixture(t, "stack.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Ecosystem != "hackage" {
		t.Errorf("Ecosystem = %q, want %q", result.Ecosystem, "hackage")
	}
	if len(result.Direct) != 3 {
		t.Fatalf("expected 3 deps, got %d", len(result.Direct))
	}
	aeson := findDep(result.Direct, "aeson")
	if aeson == nil {
		t.Fatal("missing aeson")
	}
	if aeson.Deps != nil {
		t.Error("stack deps should have nil Deps (flat list)")
	}
}

func TestDeno(t *testing.T) {
	result, err := resolve.Parse("deno", loadFixture(t, "deno.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Direct) != 2 {
		t.Fatalf("expected 2 deps, got %d", len(result.Direct))
	}
	express := findDep(result.Direct, "express")
	if express == nil {
		t.Fatal("missing express")
	}
	if express.Version != "4.18.2" {
		t.Errorf("express version = %q, want %q", express.Version, "4.18.2")
	}
}

func TestSwift(t *testing.T) {
	result, err := resolve.Parse("swift", loadFixture(t, "swift.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Direct) != 2 {
		t.Fatalf("expected 2 direct deps, got %d", len(result.Direct))
	}
	parser := findDep(result.Direct, "swift-argument-parser")
	if parser == nil {
		t.Fatal("missing swift-argument-parser")
	}
	if parser.Version != "1.2.3" {
		t.Errorf("version = %q, want %q", parser.Version, "1.2.3")
	}
	if len(parser.Deps) != 1 {
		t.Errorf("transitive deps = %d, want 1", len(parser.Deps))
	}
}

func TestUV(t *testing.T) {
	result, err := resolve.Parse("uv", loadFixture(t, "uv.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// root line "my-project v0.1.0" is parsed as depth 0, then deps at depth 0 too
	// Since the tree starts with root, we get root + 2 direct children
	// Actually the root line won't match "name vX" well... let me check
	// Root: "my-project v0.1.0" -> depth 0
	// "├── requests v2.31.0" -> depth 0
	// "│   ├── certifi v2024.12.14" -> depth 1
	// etc.
	// So buildTree will see all depth-0 items as roots
	if len(result.Direct) < 2 {
		t.Fatalf("expected at least 2 direct deps, got %d", len(result.Direct))
	}

	// Find requests (should have transitive deps)
	requests := findDep(result.Direct, "requests")
	if requests == nil {
		t.Fatal("missing requests")
	}
	if requests.Version != "2.31.0" {
		t.Errorf("requests version = %q, want %q", requests.Version, "2.31.0")
	}
	if len(requests.Deps) != 2 {
		t.Errorf("requests transitive deps = %d, want 2", len(requests.Deps))
	}
}

func TestPoetry(t *testing.T) {
	result, err := resolve.Parse("poetry", loadFixture(t, "poetry.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Direct) < 4 {
		t.Fatalf("expected at least 4 direct deps, got %d", len(result.Direct))
	}
	requests := findDep(result.Direct, "requests")
	if requests == nil {
		t.Fatal("missing requests")
	}
	if requests.Version != "2.31.0" {
		t.Errorf("requests version = %q, want %q", requests.Version, "2.31.0")
	}
	// requests has 3 sub-deps (certifi, charset-normalizer, urllib3)
	if len(requests.Deps) != 3 {
		t.Errorf("requests sub-deps = %d, want 3", len(requests.Deps))
	}
	// Cross-referenced versions
	certifi := findDep(requests.Deps, "certifi")
	if certifi == nil {
		t.Fatal("missing certifi under requests")
	}
	if certifi.Version != "2024.12.14" {
		t.Errorf("certifi version = %q, want %q", certifi.Version, "2024.12.14")
	}
}

func TestBundler(t *testing.T) {
	result, err := resolve.Parse("bundler", loadFixture(t, "bundler.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Ecosystem != "gem" {
		t.Errorf("Ecosystem = %q, want %q", result.Ecosystem, "gem")
	}
	if len(result.Direct) != 7 {
		t.Fatalf("expected 7 deps, got %d", len(result.Direct))
	}
	puma := findDep(result.Direct, "puma")
	if puma == nil {
		t.Fatal("missing puma")
	}
	if puma.Version != "6.4.0" {
		t.Errorf("puma version = %q, want %q", puma.Version, "6.4.0")
	}
	if puma.Deps != nil {
		t.Error("bundler deps should have nil Deps (flat list)")
	}
	if !strings.Contains(puma.PURL, "pkg:gem/puma@6.4.0") {
		t.Errorf("puma PURL = %q", puma.PURL)
	}
}

func TestMaven(t *testing.T) {
	result, err := resolve.Parse("maven", loadFixture(t, "maven.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Ecosystem != "maven" {
		t.Errorf("Ecosystem = %q, want %q", result.Ecosystem, "maven")
	}
	if len(result.Direct) != 3 {
		t.Fatalf("expected 3 direct deps, got %d", len(result.Direct))
	}
	guava := findDep(result.Direct, "com.google.guava:guava")
	if guava == nil {
		t.Fatal("missing guava")
	}
	if guava.Version != "32.1.3-jre" {
		t.Errorf("guava version = %q, want %q", guava.Version, "32.1.3-jre")
	}
	if len(guava.Deps) != 2 {
		t.Errorf("guava transitive deps = %d, want 2", len(guava.Deps))
	}

	junit := findDep(result.Direct, "junit:junit")
	if junit == nil {
		t.Fatal("missing junit")
	}
	if len(junit.Deps) != 1 {
		t.Errorf("junit transitive deps = %d, want 1", len(junit.Deps))
	}
}

func TestGradle(t *testing.T) {
	result, err := resolve.Parse("gradle", loadFixture(t, "gradle.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Direct) != 3 {
		t.Fatalf("expected 3 direct deps, got %d", len(result.Direct))
	}
	guava := findDep(result.Direct, "com.google.guava:guava")
	if guava == nil {
		t.Fatal("missing guava")
	}
	if guava.Version != "32.1.3-jre" {
		t.Errorf("guava version = %q, want %q", guava.Version, "32.1.3-jre")
	}
	if len(guava.Deps) != 2 {
		t.Errorf("guava transitive deps = %d, want 2", len(guava.Deps))
	}

	jackson := findDep(result.Direct, "com.fasterxml.jackson.core:jackson-databind")
	if jackson == nil {
		t.Fatal("missing jackson-databind")
	}
	if len(jackson.Deps) != 2 {
		t.Errorf("jackson transitive deps = %d, want 2", len(jackson.Deps))
	}
}

func TestComposer(t *testing.T) {
	result, err := resolve.Parse("composer", loadFixture(t, "composer.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Ecosystem != "packagist" {
		t.Errorf("Ecosystem = %q, want %q", result.Ecosystem, "packagist")
	}
	if len(result.Direct) != 2 {
		t.Fatalf("expected 2 direct deps, got %d", len(result.Direct))
	}
	laravel := findDep(result.Direct, "laravel/framework")
	if laravel == nil {
		t.Fatal("missing laravel/framework")
	}
	if laravel.Version != "v10.38.1" {
		t.Errorf("laravel version = %q, want %q", laravel.Version, "v10.38.1")
	}
	if len(laravel.Deps) != 3 {
		t.Errorf("laravel transitive deps = %d, want 3", len(laravel.Deps))
	}
	// Check nested guzzle deps
	guzzle := findDep(laravel.Deps, "guzzlehttp/guzzle")
	if guzzle == nil {
		t.Fatal("missing guzzlehttp/guzzle")
	}
	if len(guzzle.Deps) != 2 {
		t.Errorf("guzzle transitive deps = %d, want 2", len(guzzle.Deps))
	}
}

func TestNuget(t *testing.T) {
	result, err := resolve.Parse("nuget", loadFixture(t, "nuget.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Direct) != 4 {
		t.Fatalf("expected 4 deps, got %d", len(result.Direct))
	}
	newtonsoft := findDep(result.Direct, "Newtonsoft.Json")
	if newtonsoft == nil {
		t.Fatal("missing Newtonsoft.Json")
	}
	if newtonsoft.Version != "13.0.3" {
		t.Errorf("version = %q, want %q", newtonsoft.Version, "13.0.3")
	}
	// Flat: Deps nil
	if newtonsoft.Deps != nil {
		t.Error("nuget deps should have nil Deps (flat list)")
	}
}

func TestPub(t *testing.T) {
	result, err := resolve.Parse("pub", loadFixture(t, "pub.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Direct) < 3 {
		t.Fatalf("expected at least 3 direct deps, got %d", len(result.Direct))
	}
	http := findDep(result.Direct, "http")
	if http == nil {
		t.Fatal("missing http")
	}
	if http.Version != "1.1.2" {
		t.Errorf("http version = %q, want %q", http.Version, "1.1.2")
	}
	if len(http.Deps) != 2 {
		t.Errorf("http transitive deps = %d, want 2", len(http.Deps))
	}
}

func TestMix(t *testing.T) {
	result, err := resolve.Parse("mix", loadFixture(t, "mix.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Ecosystem != "hex" {
		t.Errorf("Ecosystem = %q, want %q", result.Ecosystem, "hex")
	}
	if len(result.Direct) < 3 {
		t.Fatalf("expected at least 3 direct deps, got %d", len(result.Direct))
	}
	phoenix := findDep(result.Direct, "phoenix")
	if phoenix == nil {
		t.Fatal("missing phoenix")
	}
	if phoenix.Version != "1.7.10" {
		t.Errorf("phoenix version = %q, want %q", phoenix.Version, "1.7.10")
	}
	if len(phoenix.Deps) != 2 {
		t.Errorf("phoenix transitive deps = %d, want 2", len(phoenix.Deps))
	}
}

func TestRebar3(t *testing.T) {
	result, err := resolve.Parse("rebar3", loadFixture(t, "rebar3.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Ecosystem != "hex" {
		t.Errorf("Ecosystem = %q, want %q", result.Ecosystem, "hex")
	}
	if len(result.Direct) < 2 {
		t.Fatalf("expected at least 2 direct deps, got %d", len(result.Direct))
	}
	cowboy := findDep(result.Direct, "cowboy")
	if cowboy == nil {
		t.Fatal("missing cowboy")
	}
	if cowboy.Version != "2.10.0" {
		t.Errorf("cowboy version = %q, want %q", cowboy.Version, "2.10.0")
	}
	if len(cowboy.Deps) != 2 {
		t.Errorf("cowboy transitive deps = %d, want 2", len(cowboy.Deps))
	}
}

func TestLein(t *testing.T) {
	result, err := resolve.Parse("lein", loadFixture(t, "lein.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Ecosystem != "clojars" {
		t.Errorf("Ecosystem = %q, want %q", result.Ecosystem, "clojars")
	}
	if len(result.Direct) != 3 {
		t.Fatalf("expected 3 direct deps, got %d", len(result.Direct))
	}
	clojure := findDep(result.Direct, "org.clojure/clojure")
	if clojure == nil {
		t.Fatal("missing org.clojure/clojure")
	}
	if clojure.Version != "1.11.1" {
		t.Errorf("clojure version = %q, want %q", clojure.Version, "1.11.1")
	}
	// clojure has 1 sub-dep
	if len(clojure.Deps) != 1 {
		t.Errorf("clojure transitive deps = %d, want 1", len(clojure.Deps))
	}
}

func TestConan(t *testing.T) {
	result, err := resolve.Parse("conan", loadFixture(t, "conan.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Direct) != 3 {
		t.Fatalf("expected 3 deps, got %d", len(result.Direct))
	}
	boost := findDep(result.Direct, "boost")
	if boost == nil {
		t.Fatal("missing boost")
	}
	if boost.Version != "1.83.0" {
		t.Errorf("boost version = %q, want %q", boost.Version, "1.83.0")
	}
	// Flat: Deps nil
	if boost.Deps != nil {
		t.Error("conan deps should have nil Deps (flat list)")
	}
}

func TestHelm(t *testing.T) {
	result, err := resolve.Parse("helm", loadFixture(t, "helm.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Direct) != 3 {
		t.Fatalf("expected 3 deps, got %d", len(result.Direct))
	}
	postgres := findDep(result.Direct, "postgresql")
	if postgres == nil {
		t.Fatal("missing postgresql")
	}
	if postgres.Version != "12.1.9" {
		t.Errorf("postgresql version = %q, want %q", postgres.Version, "12.1.9")
	}
	if postgres.Deps != nil {
		t.Error("helm deps should have nil Deps (flat list)")
	}
}

func TestParseEmptyInput(t *testing.T) {
	_, err := resolve.Parse("npm", []byte(""))
	if err == nil {
		t.Error("expected error for empty input")
	}
}
