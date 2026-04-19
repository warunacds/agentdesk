package skills

import (
	"testing"
)

// ---------------------------------------------------------------------------
// BuiltinTemplates
// ---------------------------------------------------------------------------

func TestBuiltinTemplates_NotEmpty(t *testing.T) {
	if len(BuiltinTemplates) == 0 {
		t.Fatal("BuiltinTemplates is empty — expected at least one template")
	}
}

func TestBuiltinTemplates_RequiredFields(t *testing.T) {
	for _, tmpl := range BuiltinTemplates {
		t.Run(tmpl.ID, func(t *testing.T) {
			if tmpl.ID == "" {
				t.Error("template has empty ID")
			}
			if tmpl.Name == "" {
				t.Errorf("template %q has empty Name", tmpl.ID)
			}
			if tmpl.Description == "" {
				t.Errorf("template %q has empty Description", tmpl.ID)
			}
			if tmpl.Content == "" {
				t.Errorf("template %q has empty Content", tmpl.ID)
			}
			if tmpl.Tool == "" {
				t.Errorf("template %q has empty Tool", tmpl.ID)
			}
			if tmpl.Category == "" {
				t.Errorf("template %q has empty Category", tmpl.ID)
			}
		})
	}
}

func TestBuiltinTemplates_UniqueIDs(t *testing.T) {
	seen := make(map[string]bool)
	for _, tmpl := range BuiltinTemplates {
		if seen[tmpl.ID] {
			t.Errorf("duplicate template ID: %q", tmpl.ID)
		}
		seen[tmpl.ID] = true
	}
}

// ---------------------------------------------------------------------------
// FindTemplate
// ---------------------------------------------------------------------------

func TestFindTemplate(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		wantNil  bool
		wantTool ToolType
		wantCat  string
	}{
		{
			name:     "claude agent react",
			id:       "claude-agent-react",
			wantNil:  false,
			wantTool: ToolClaudeCode,
			wantCat:  "agent",
		},
		{
			name:     "claude agent python",
			id:       "claude-agent-python",
			wantNil:  false,
			wantTool: ToolClaudeCode,
			wantCat:  "agent",
		},
		{
			name:     "claude agent go",
			id:       "claude-agent-go",
			wantNil:  false,
			wantTool: ToolClaudeCode,
			wantCat:  "agent",
		},
		{
			name:     "claude agent general",
			id:       "claude-agent-general",
			wantNil:  false,
			wantTool: ToolClaudeCode,
			wantCat:  "agent",
		},
		{
			name:     "cursor rule nextjs",
			id:       "cursor-rule-nextjs",
			wantNil:  false,
			wantTool: ToolCursor,
			wantCat:  "rule",
		},
		{
			name:     "cursor rule typescript",
			id:       "cursor-rule-typescript",
			wantNil:  false,
			wantTool: ToolCursor,
			wantCat:  "rule",
		},
		{
			name:     "cursor rule python",
			id:       "cursor-rule-python",
			wantNil:  false,
			wantTool: ToolCursor,
			wantCat:  "rule",
		},
		{
			name:     "gemini agent",
			id:       "gemini-agent",
			wantNil:  false,
			wantTool: ToolGeminiCLI,
			wantCat:  "agent",
		},
		{
			name:     "kiro steering",
			id:       "kiro-steering",
			wantNil:  false,
			wantTool: ToolKiro,
			wantCat:  "rule",
		},
		{
			name:     "codex agent",
			id:       "codex-agent",
			wantNil:  false,
			wantTool: ToolCodex,
			wantCat:  "agent",
		},
		{
			name:    "nonexistent returns nil",
			id:      "does-not-exist",
			wantNil: true,
		},
		{
			name:    "empty string returns nil",
			id:      "",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := FindTemplate(tt.id)
			if tt.wantNil {
				if tmpl != nil {
					t.Errorf("FindTemplate(%q) = %v, want nil", tt.id, tmpl)
				}
				return
			}

			if tmpl == nil {
				t.Fatalf("FindTemplate(%q) = nil, want non-nil", tt.id)
			}
			if tmpl.ID != tt.id {
				t.Errorf("FindTemplate(%q).ID = %q", tt.id, tmpl.ID)
			}
			if tmpl.Tool != tt.wantTool {
				t.Errorf("FindTemplate(%q).Tool = %q, want %q", tt.id, tmpl.Tool, tt.wantTool)
			}
			if tmpl.Category != tt.wantCat {
				t.Errorf("FindTemplate(%q).Category = %q, want %q", tt.id, tmpl.Category, tt.wantCat)
			}
		})
	}
}

// TestFindTemplate_ReturnsPointerToOriginal verifies that FindTemplate returns
// a pointer into the BuiltinTemplates slice, not a copy.
func TestFindTemplate_ReturnsPointerToOriginal(t *testing.T) {
	tmpl := FindTemplate("claude-agent-react")
	if tmpl == nil {
		t.Fatal("FindTemplate returned nil")
	}

	// Find the same entry in the slice and compare addresses.
	for i := range BuiltinTemplates {
		if BuiltinTemplates[i].ID == "claude-agent-react" {
			if tmpl != &BuiltinTemplates[i] {
				t.Error("FindTemplate did not return a pointer into BuiltinTemplates")
			}
			return
		}
	}
	t.Error("claude-agent-react not found in BuiltinTemplates")
}
