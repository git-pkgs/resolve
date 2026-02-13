package parsers

import (
	"regexp"
	"strings"

	"github.com/git-pkgs/resolve"
)

// rebar3PkgRe matches "name─version" or "name─version (hex package)".
var rebar3PkgRe = regexp.MustCompile(`^(\S+?)─(\S+?)(?:\s|$)`)

// parseRebar3 parses output from `rebar3 tree`.
// Lines like "├─ name─version (hex package)" with single-width dashes.
func parseRebar3(data []byte) ([]*resolve.Dep, error) {
	lines := strings.Split(string(data), "\n")
	opts := resolve.TreeOptions{
		Prefixes:      []string{"├─ ", "└─ "},
		Continuations: []string{"│  ", "   "},
	}
	treeLines := resolve.ParseTreeLines(lines, opts)

	return resolve.BuildTree(treeLines, "hex", func(content string) (string, string, bool) {
		m := rebar3PkgRe.FindStringSubmatch(content)
		if m == nil {
			return "", "", false
		}
		return m[1], m[2], true
	}), nil
}

func init() {
	resolve.Register("rebar3", "hex", parseRebar3)
}
