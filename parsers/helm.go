package parsers

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/git-pkgs/resolve"
)

// parseHelm parses output from `helm dependency list`.
// Format: tab-separated table with header: NAME VERSION REPOSITORY STATUS
func parseHelm(data []byte) ([]*resolve.Dep, error) {
	var deps []*resolve.Dep
	scanner := bufio.NewScanner(bytes.NewReader(data))
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first {
			first = false
			continue // skip header
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := fields[0]
		version := fields[1]
		deps = append(deps, &resolve.Dep{
			PURL:    resolve.MakePURL("helm", name, version),
			Name:    name,
			Version: version,
		})
	}
	return deps, nil
}

func init() {
	resolve.Register("helm", "helm", parseHelm)
}
