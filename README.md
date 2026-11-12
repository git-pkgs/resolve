# resolve

Parses raw package manager CLI output into a normalized dependency graph with [PURLs](https://github.com/package-url/purl-spec).

Takes the output bytes from a manager's resolve command (e.g. `npm ls --json`, `go mod graph`, `uv tree`) and returns a structured `Result` with the dependency tree and PURL for each package.

## Installation

```sh
go get github.com/git-pkgs/resolve
```

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

`Parse` and `ParseDir` are the entry points: `Parse` reads command output and `ParseDir` reads files from a project directory (see [Directory parsers](#directory-parsers)). Each dispatches to the correct parser based on the manager name and returns `ErrUnsupportedManager` for unknown managers.

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
| apm | apm | Lockfile graph |

## Directory parsers

Some managers record their resolved graph in a lockfile rather than in command output. `ParseDir(manager, dir)` reads those files from a project directory.

APM reads `apm.lock.yaml`, or the legacy `apm.lock` when the current file is absent, without running `apm` or reading `apm_modules/`. Package identities come from the APM parser in [manifests](https://github.com/git-pkgs/manifests), so Git dependencies keep their commit and repository subpath, while local and registry sources have no PURL. Edges come from each entry's `depth` and `resolved_by`. As in APM's own `apm deps tree`, a parent is the single entry at the previous depth whose lock key or `repo_url` matches `resolved_by`. When several entries match, such as two subpackages of one repository, `ParseDir` returns an ambiguous-parent error instead of picking one. Missing parents and duplicate entries are errors too.

## License

[MIT](LICENSE).
