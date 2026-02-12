package resolve

import (
	"testing"
)

func TestParseTreeLinesBoxDrawing(t *testing.T) {
	lines := []string{
		"root-package",
		"├── dep-a v1.0.0",
		"│   ├── sub-a1 v0.1.0",
		"│   └── sub-a2 v0.2.0",
		"└── dep-b v2.0.0",
	}
	opts := BoxDrawingOptions()
	result := ParseTreeLines(lines, opts)

	if len(result) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(result))
	}

	tests := []struct {
		depth   int
		content string
	}{
		{0, "root-package"},
		{0, "dep-a v1.0.0"},
		{1, "sub-a1 v0.1.0"},
		{1, "sub-a2 v0.2.0"},
		{0, "dep-b v2.0.0"},
	}

	for i, tt := range tests {
		if result[i].Depth != tt.depth {
			t.Errorf("line %d: depth = %d, want %d", i, result[i].Depth, tt.depth)
		}
		if result[i].Content != tt.content {
			t.Errorf("line %d: content = %q, want %q", i, result[i].Content, tt.content)
		}
	}
}

func TestParseTreeLinesSkipsEmptyLines(t *testing.T) {
	lines := []string{"", "foo", "", "bar", ""}
	opts := BoxDrawingOptions()
	result := ParseTreeLines(lines, opts)
	if len(result) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(result))
	}
}

func TestParseTreeLinesRebar3Style(t *testing.T) {
	lines := []string{
		"root",
		"├─ cowboy─2.10.0 (hex package)",
		"│  └─ ranch─1.8.0 (hex package)",
	}
	opts := TreeOptions{
		Prefixes:      []string{"├─ ", "└─ "},
		Continuations: []string{"│  ", "   "},
	}
	result := ParseTreeLines(lines, opts)

	if len(result) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(result))
	}
	if result[2].Depth != 1 {
		t.Errorf("sub-dep depth = %d, want 1", result[2].Depth)
	}
}

func TestBuildTree(t *testing.T) {
	lines := []TreeLine{
		{Depth: 0, Content: "a 1.0"},
		{Depth: 1, Content: "b 2.0"},
		{Depth: 1, Content: "c 3.0"},
		{Depth: 0, Content: "d 4.0"},
	}

	deps := buildTree(lines, "npm", func(content string) (string, string, bool) {
		parts := splitFields(content)
		if len(parts) < 2 {
			return "", "", false
		}
		return parts[0], parts[1], true
	})

	if len(deps) != 2 {
		t.Fatalf("expected 2 root deps, got %d", len(deps))
	}
	if deps[0].Name != "a" {
		t.Errorf("first dep name = %q, want %q", deps[0].Name, "a")
	}
	if len(deps[0].Deps) != 2 {
		t.Errorf("first dep children = %d, want 2", len(deps[0].Deps))
	}
	if deps[1].Name != "d" {
		t.Errorf("second dep name = %q, want %q", deps[1].Name, "d")
	}
	if len(deps[1].Deps) != 0 {
		t.Errorf("second dep children = %d, want 0", len(deps[1].Deps))
	}
	// Tree structure means Deps is non-nil
	if deps[1].Deps == nil {
		t.Error("Deps should be non-nil empty slice for tree managers")
	}
}

func splitFields(s string) []string {
	var fields []string
	field := ""
	for _, ch := range s {
		if ch == ' ' || ch == '\t' {
			if field != "" {
				fields = append(fields, field)
				field = ""
			}
		} else {
			field += string(ch)
		}
	}
	if field != "" {
		fields = append(fields, field)
	}
	return fields
}
