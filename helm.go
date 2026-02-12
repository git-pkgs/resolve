package resolve

import (
	"bufio"
	"bytes"
	"strings"
)

// parseHelm parses output from `helm dependency list`.
// Format: tab-separated table with header: NAME VERSION REPOSITORY STATUS
func parseHelm(data []byte) ([]*Dep, error) {
	var deps []*Dep
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
		deps = append(deps, &Dep{
			PURL:    makePURL("helm", name, version),
			Name:    name,
			Version: version,
		})
	}
	return deps, nil
}
