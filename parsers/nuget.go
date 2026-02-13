package parsers

import (
	"bufio"
	"bytes"
	"regexp"
	"strings"

	"github.com/git-pkgs/resolve"
)

// nugetPkgRe matches "> PackageName  version" or "> PackageName  (requested) resolved".
var nugetPkgRe = regexp.MustCompile(`>\s+(\S+)\s+(?:\([^)]+\)\s+)?(\S+)`)

// parseNuget parses output from `dotnet list package --include-transitive`.
func parseNuget(data []byte) ([]*resolve.Dep, error) {
	var deps []*resolve.Dep
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, ">") {
			continue
		}
		m := nugetPkgRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := m[1]
		version := m[2]
		deps = append(deps, &resolve.Dep{
			PURL:    resolve.MakePURL("nuget", name, version),
			Name:    name,
			Version: version,
		})
	}
	return deps, nil
}

func init() {
	resolve.Register("nuget", "nuget", parseNuget)
}
