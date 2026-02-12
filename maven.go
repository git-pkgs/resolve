package resolve

import (
	"bufio"
	"bytes"
	"strings"
)

// parseMaven parses output from `mvn dependency:tree`.
// Lines prefixed with [INFO] then tree markers (+- | \-).
// Package format: group:artifact:type:version:scope
func parseMaven(data []byte) ([]*Dep, error) {
	var treeLines []TreeLine
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "[INFO] ") {
			continue
		}
		line = strings.TrimPrefix(line, "[INFO] ")

		// Skip non-tree lines
		if line == "" || strings.HasPrefix(line, "---") || strings.HasPrefix(line, "Building") ||
			strings.HasPrefix(line, "Scanning") || strings.HasPrefix(line, "Download") ||
			strings.HasPrefix(line, "Artifact") {
			continue
		}

		// Only process lines that have tree markers (skip root project line)
		hasMarker := false
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
		for _, prefix := range []string{"+- ", "\\- "} {
			if strings.HasPrefix(remaining, prefix) {
				remaining = remaining[len(prefix):]
				hasMarker = true
				break
			}
		}
		if depth > 0 {
			hasMarker = true
		}
		if !hasMarker {
			continue
		}

		// Parse maven coordinate: group:artifact:type:version[:scope]
		parts := strings.Split(remaining, ":")
		if len(parts) < 4 {
			continue
		}
		group := parts[0]
		artifact := parts[1]
		version := parts[3]
		name := group + ":" + artifact

		treeLines = append(treeLines, TreeLine{Depth: depth, Content: name + "\t" + version})
	}

	return buildTree(treeLines, "maven", func(content string) (string, string, bool) {
		parts := strings.SplitN(content, "\t", 2)
		if len(parts) != 2 {
			return "", "", false
		}
		return parts[0], parts[1], true
	}), nil
}
