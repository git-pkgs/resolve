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

		// Detect configuration headers like "compileClasspath - ..."
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "|") &&
			!strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "\\") &&
			strings.Contains(line, " - ") {
			if configDone {
				continue // only use first configuration
			}
			inConfig = true
			continue
		}

		if !inConfig {
			continue
		}

		// Empty line ends a configuration
		if strings.TrimSpace(line) == "" {
			inConfig = false
			configDone = true
			continue
		}

		// Skip duplicate markers
		if strings.Contains(line, "(*)") {
			continue
		}
		// Skip constraint markers
		if strings.Contains(line, "(c)") {
			continue
		}

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

		remaining = strings.TrimSpace(remaining)

		// Parse gradle coordinate: group:name:version [-> resolvedVersion]
		parts := strings.Split(remaining, ":")
		if len(parts) < 3 {
			continue
		}
		group := parts[0]
		artifact := parts[1]
		versionPart := parts[2]

		// Handle version resolution arrows: "1.0 -> 2.0"
		version := versionPart
		if idx := strings.Index(versionPart, " -> "); idx >= 0 {
			version = versionPart[idx+4:]
		}
		version = strings.TrimSpace(version)

		name := group + ":" + artifact
		treeLines = append(treeLines, resolve.TreeLine{Depth: depth, Content: name + "\t" + version})
	}

	return resolve.BuildTree(treeLines, "maven", func(content string) (string, string, bool) {
		parts := strings.SplitN(content, "\t", 2)
		if len(parts) != 2 {
			return "", "", false
		}
		return parts[0], parts[1], true
	}), nil
}

func init() {
	resolve.Register("gradle", "maven", parseGradle)
}
