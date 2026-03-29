package parsers

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/git-pkgs/resolve"
)

// parseMaven parses output from `mvn dependency:tree`.
// Lines prefixed with [INFO] then tree markers (+- | \-).
// Package format: group:artifact:type:version:scope
func parseMaven(data []byte) ([]*resolve.Dep, error) {
	var treeLines []resolve.TreeLine
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "[INFO] ") {
			continue
		}
		line = strings.TrimPrefix(line, "[INFO] ")

		if isMavenNonTreeLine(line) {
			continue
		}

		depth, remaining, hasMarker := parseMavenTreeDepth(line)
		if !hasMarker {
			continue
		}

		tl, ok := parseMavenCoordinate(remaining, depth)
		if ok {
			treeLines = append(treeLines, tl)
		}
	}

	return resolve.BuildTree(treeLines, "maven", resolve.TabContentParser), nil
}

func isMavenNonTreeLine(line string) bool {
	return line == "" || strings.HasPrefix(line, "---") || strings.HasPrefix(line, "Building") ||
		strings.HasPrefix(line, "Scanning") || strings.HasPrefix(line, "Download") ||
		strings.HasPrefix(line, "Artifact")
}

func parseMavenTreeDepth(line string) (int, string, bool) {
	depth := 0
	remaining := line
	for {
		found := false
		for _, cont := range []string{"|  ", "   "} {
			if strings.HasPrefix(remaining, cont) {
				depth++
				remaining = remaining[len(cont):]
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	hasMarker := depth > 0
	for _, prefix := range []string{"+- ", "\\- "} {
		if strings.HasPrefix(remaining, prefix) {
			remaining = remaining[len(prefix):]
			hasMarker = true
			break
		}
	}
	return depth, remaining, hasMarker
}

func parseMavenCoordinate(s string, depth int) (resolve.TreeLine, bool) {
	parts := strings.Split(s, ":")
	if len(parts) < 4 { //nolint:mnd // group:artifact:type:version
		return resolve.TreeLine{}, false
	}
	name := parts[0] + ":" + parts[1]
	version := parts[3]
	return resolve.TreeLine{Depth: depth, Content: name + "\t" + version}, true
}

func init() {
	resolve.Register("maven", "maven", parseMaven)
}
