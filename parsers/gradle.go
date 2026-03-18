package parsers

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/git-pkgs/resolve"
)

// parseGradle parses output from `gradle dependencies`.
// Multiple configurations. Tree markers (+--- |    \---).
// Package format: group:name:version. Lines with (*) are duplicates to skip.
func parseGradle(data []byte) ([]*resolve.Dep, error) {
	var treeLines []resolve.TreeLine
	scanner := bufio.NewScanner(bytes.NewReader(data))
	inConfig := false
	configDone := false

	for scanner.Scan() {
		line := scanner.Text()

		if isGradleConfigHeader(line) {
			if configDone {
				continue
			}
			inConfig = true
			continue
		}

		if !inConfig {
			continue
		}

		if strings.TrimSpace(line) == "" {
			inConfig = false
			configDone = true
			continue
		}

		if strings.Contains(line, "(*)") || strings.Contains(line, "(c)") {
			continue
		}

		depth, remaining := parseGradleTreeDepth(line)
		tl, ok := parseGradleCoordinate(remaining, depth)
		if ok {
			treeLines = append(treeLines, tl)
		}
	}

	return resolve.BuildTree(treeLines, "maven", resolve.TabContentParser), nil
}

func isGradleConfigHeader(line string) bool {
	return !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "|") &&
		!strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "\\") &&
		strings.Contains(line, " - ")
}

func parseGradleTreeDepth(line string) (int, string) {
	depth := 0
	remaining := line
	for {
		found := false
		for _, cont := range []string{"|    ", "     "} {
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
	for _, prefix := range []string{"+--- ", "\\--- "} {
		if strings.HasPrefix(remaining, prefix) {
			remaining = remaining[len(prefix):]
			break
		}
	}
	return depth, strings.TrimSpace(remaining)
}

func parseGradleCoordinate(s string, depth int) (resolve.TreeLine, bool) {
	parts := strings.Split(s, ":")
	if len(parts) < 3 { //nolint:mnd // group:name:version
		return resolve.TreeLine{}, false
	}
	group := parts[0]
	artifact := parts[1]
	versionPart := parts[2]

	version := versionPart
	if _, resolved, ok := strings.Cut(versionPart, " -> "); ok {
		version = resolved
	}
	version = strings.TrimSpace(version)

	name := group + ":" + artifact
	return resolve.TreeLine{Depth: depth, Content: name + "\t" + version}, true
}

func init() {
	resolve.Register("gradle", "maven", parseGradle)
}
