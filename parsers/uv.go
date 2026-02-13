package parsers

import (
	"regexp"
	"strings"

	"github.com/git-pkgs/resolve"
)

// uvPkgRe matches lines like "certifi v2024.12.14" or "requests v2.31.0".
var uvPkgRe = regexp.MustCompile(`^(\S+)\s+v(.+)$`)

// parseUV parses output from `uv tree`.
func parseUV(data []byte) ([]*resolve.Dep, error) {
	lines := strings.Split(string(data), "\n")
	opts := resolve.BoxDrawingOptions()
	treeLines := resolve.ParseTreeLines(lines, opts)

	return resolve.BuildTree(treeLines, "pypi", func(content string) (string, string, bool) {
		m := uvPkgRe.FindStringSubmatch(content)
		if m == nil {
			return "", "", false
		}
		return m[1], m[2], true
	}), nil
}

func init() {
	resolve.Register("uv", "pypi", parseUV)
}
