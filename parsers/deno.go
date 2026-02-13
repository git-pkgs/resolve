package parsers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/git-pkgs/resolve"
)

// parseDeno parses output from `deno info --json`.
func parseDeno(data []byte) ([]*resolve.Dep, error) {
	var output struct {
		Modules []struct {
			Specifier    string `json:"specifier"`
			Dependencies []struct {
				Specifier string `json:"specifier"`
				Code      *struct {
					Specifier string `json:"specifier"`
				} `json:"code"`
			} `json:"dependencies"`
		} `json:"modules"`
	}

	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("parsing deno output: %w", err)
	}

	// Collect unique packages
	seen := make(map[string]bool)
	var deps []*resolve.Dep
	for _, mod := range output.Modules {
		name, version := parseDenoSpecifier(mod.Specifier)
		if name == "" || seen[name+"@"+version] {
			continue
		}
		seen[name+"@"+version] = true
		deps = append(deps, &resolve.Dep{
			PURL:    resolve.MakePURL("deno", name, version),
			Name:    name,
			Version: version,
		})
	}
	return deps, nil
}

func parseDenoSpecifier(spec string) (string, string) {
	// npm:express@4.18.2 or npm:@scope/pkg@1.0.0
	if strings.HasPrefix(spec, "npm:") {
		rest := spec[4:]
		if idx := strings.LastIndex(rest, "@"); idx > 0 {
			return rest[:idx], rest[idx+1:]
		}
		return rest, ""
	}

	// jsr:@std/path@0.200.0
	if strings.HasPrefix(spec, "jsr:") {
		rest := spec[4:]
		if idx := strings.LastIndex(rest, "@"); idx > 0 {
			return rest[:idx], rest[idx+1:]
		}
		return rest, ""
	}

	// https://deno.land/std@0.200.0/path/mod.ts -> name="std", version="0.200.0"
	// https://deno.land/x/oak@12.0.0/mod.ts -> name="oak", version="12.0.0"
	if strings.HasPrefix(spec, "https://deno.land/") {
		rest := spec[len("https://deno.land/"):]
		// Strip "std/" or "x/" prefix if followed by more path
		if strings.HasPrefix(rest, "x/") {
			rest = rest[2:]
		}
		// Now rest is like "std@0.200.0/path/mod.ts" or "oak@12.0.0/mod.ts"
		if idx := strings.Index(rest, "@"); idx > 0 {
			name := rest[:idx]
			versionAndPath := rest[idx+1:]
			version := versionAndPath
			if slashIdx := strings.Index(versionAndPath, "/"); slashIdx > 0 {
				version = versionAndPath[:slashIdx]
			}
			return name, version
		}
	}

	return "", ""
}

func init() {
	resolve.Register("deno", "deno", parseDeno)
}
