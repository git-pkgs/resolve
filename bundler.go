package resolve

import (
	"bufio"
	"bytes"
	"regexp"
)

// bundlerLineRe matches lines like "  * name (version)" or "  * name (version hash)".
var bundlerLineRe = regexp.MustCompile(`^\s+\*\s+(\S+)\s+\(([^)\s]+)`)

// parseBundler parses output from `bundle list`.
func parseBundler(data []byte) ([]*Dep, error) {
	var deps []*Dep
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		m := bundlerLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := m[1]
		version := m[2]
		deps = append(deps, &Dep{
			PURL:    makePURL("gem", name, version),
			Name:    name,
			Version: version,
		})
	}
	return deps, nil
}
