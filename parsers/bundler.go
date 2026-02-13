package parsers

import (
	"bufio"
	"bytes"
	"regexp"

	"github.com/git-pkgs/resolve"
)

// bundlerLineRe matches lines like "  * name (version)" or "  * name (version hash)".
var bundlerLineRe = regexp.MustCompile(`^\s+\*\s+(\S+)\s+\(([^)\s]+)`)

// parseBundler parses output from `bundle list`.
func parseBundler(data []byte) ([]*resolve.Dep, error) {
	var deps []*resolve.Dep
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		m := bundlerLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := m[1]
		version := m[2]
		deps = append(deps, &resolve.Dep{
			PURL:    resolve.MakePURL("gem", name, version),
			Name:    name,
			Version: version,
		})
	}
	return deps, nil
}

func init() {
	resolve.Register("bundler", "gem", parseBundler)
}
