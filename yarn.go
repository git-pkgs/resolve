package resolve

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// parseYarn parses output from `yarn list --json`.
// NDJSON format where one line has {"type":"tree","data":{"trees":[...]}}.
func parseYarn(data []byte) ([]*Dep, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Bytes()
		var entry struct {
			Type string `json:"type"`
			Data struct {
				Trees []yarnTree `json:"trees"`
			} `json:"data"`
		}
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry.Type == "tree" {
			return walkYarnTrees(entry.Data.Trees), nil
		}
	}
	return nil, fmt.Errorf("no tree entry found in yarn output")
}

type yarnTree struct {
	Name     string     `json:"name"`
	Children []yarnTree `json:"children"`
}

func walkYarnTrees(trees []yarnTree) []*Dep {
	var result []*Dep
	for _, tree := range trees {
		name, version := parseYarnName(tree.Name)
		if name == "" {
			continue
		}
		dep := &Dep{
			PURL:    makePURL("npm", name, version),
			Name:    name,
			Version: version,
			Deps:    []*Dep{},
		}
		if len(tree.Children) > 0 {
			dep.Deps = walkYarnTrees(tree.Children)
		}
		result = append(result, dep)
	}
	return result
}

// parseYarnName splits "name@version" into name and version.
// Handles scoped packages like "@scope/name@version".
func parseYarnName(s string) (string, string) {
	// For scoped packages, the @ for version is the last @
	idx := strings.LastIndex(s, "@")
	if idx <= 0 {
		return s, ""
	}
	return s[:idx], s[idx+1:]
}
