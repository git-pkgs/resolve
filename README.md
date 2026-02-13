# resolve

Parses raw package manager CLI output into a normalized dependency graph with [PURLs](https://github.com/package-url/purl-spec).

Takes the output bytes from a manager's resolve command (e.g. `npm ls --json`, `go mod graph`, `uv tree`) and returns a structured `Result` with the dependency tree and PURL for each package.

## Usage

```go
import (
	"github.com/git-pkgs/resolve"
	_ "github.com/git-pkgs/resolve/parsers" // register all parsers
)

output, _ := exec.Command("npm", "ls", "--depth", "Infinity", "--json", "--long").Output()

result, err := resolve.Parse("npm", output)
// result.Manager   == "npm"
// result.Ecosystem == "npm"
// result.Direct    == []*Dep{ {PURL: "pkg:npm/express@4.18.2", Name: "express", Version: "4.18.2", Deps: [...]}, ... }
```

`Parse` is the only entry point. It dispatches to the correct parser based on the manager name and returns `ErrUnsupportedManager` for unknown managers.

Each `Dep` includes the ecosystem-native package name, resolved version, a PURL string, and a `Deps` slice for transitive dependencies. `Deps` is nil for managers that only produce flat lists (pip, conda, bundler, helm, etc.) and non-nil for managers that provide tree structure.

## Supported managers

| Manager | Ecosystem | Output format |
|---------|-----------|---------------|
| npm | npm | JSON tree |
| pnpm | npm | JSON tree |
| yarn | npm | NDJSON tree |
| bun | npm | Text tree |
| cargo | cargo | JSON graph |
| gomod | golang | Edge list |
| pip | pypi | JSON flat |
| uv | pypi | Text tree |
| poetry | pypi | Text tree |
| conda | conda | JSON flat |
| bundler | gem | Text flat |
| maven | maven | Text tree |
| gradle | maven | Text tree |
| composer | packagist | Text tree |
| nuget | nuget | Tabular |
| swift | swift | JSON tree |
| pub | pub | Text tree |
| mix | hex | Text tree |
| rebar3 | hex | Text tree |
| stack | hackage | JSON flat |
| lein | clojars | Text tree |
| conan | conan | Custom |
| deno | deno | JSON flat |
| helm | helm | Tabular |
