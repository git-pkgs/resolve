package parsers

import (
	"bufio"
	"bytes"
	"regexp"
	"strings"

	"github.com/git-pkgs/resolve"
)

// conanRefRe matches package reference lines like "name/version" or "name/version@user/channel".
var conanRefRe = regexp.MustCompile(`^(\S+)/(\S+?)(?:@|$)`)

// parseConan parses output from `conan info .`.
// Multi-line blocks per package, each starting with a package reference line.
func parseConan(data []byte) ([]*resolve.Dep, error) {
	var deps []*resolve.Dep
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()
		// Skip indented lines (key-value metadata)
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip lines that look like headers or paths
		if strings.HasPrefix(line, "conanfile") || strings.HasPrefix(line, "[") {
			continue
		}

		m := conanRefRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := m[1]
		version := m[2]
		deps = append(deps, &resolve.Dep{
			PURL:    resolve.MakePURL("conan", name, version),
			Name:    name,
			Version: version,
		})
	}
	return deps, nil
}

func init() {
	resolve.Register("conan", "conan", parseConan)
}
