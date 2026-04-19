package skills

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// ValidateSkill
// ---------------------------------------------------------------------------

func TestValidateSkill(t *testing.T) {
	tests := []struct {
		name      string
		skill     Skill
		wantTypes []string // expected warning types; nil means no warnings
	}{
		{
			name:      "empty file by size",
			skill:     Skill{Path: "/test/empty.md", Content: "", Size: 0},
			wantTypes: []string{"empty-file"},
		},
		{
			name:      "whitespace only content",
			skill:     Skill{Path: "/test/blank.md", Content: "   \n  \t  ", Size: 10},
			wantTypes: []string{"empty-file"},
		},
		{
			name: "large file",
			skill: Skill{
				Path:        "/test/huge.md",
				Content:     strings.Repeat("x", 101*1024),
				Size:        101 * 1024,
				Frontmatter: map[string]string{"name": "big", "description": "large file"},
			},
			wantTypes: []string{"large-file"},
		},
		{
			name: "malformed frontmatter unclosed",
			skill: Skill{
				Path:        "/test/broken.md",
				Content:     "---\nname: test\nno closing fence",
				Size:        30,
				Frontmatter: map[string]string{"name": "test"},
			},
			wantTypes: []string{"malformed-frontmatter"},
		},
		{
			name: "missing name in frontmatter",
			skill: Skill{
				Path:        "/test/noname.md",
				Content:     "---\ndescription: test\n---\nbody",
				Size:        30,
				Frontmatter: map[string]string{"description": "test"},
			},
			wantTypes: []string{"missing-name"},
		},
		{
			name: "missing description in frontmatter",
			skill: Skill{
				Path:        "/test/nodesc.md",
				Content:     "---\nname: test\n---\nbody",
				Size:        22,
				Frontmatter: map[string]string{"name": "test"},
			},
			wantTypes: []string{"missing-description"},
		},
		{
			name: "missing both name and description",
			skill: Skill{
				Path:        "/test/bare.md",
				Content:     "---\nfoo: bar\n---\nbody",
				Size:        20,
				Frontmatter: map[string]string{"foo": "bar"},
			},
			wantTypes: []string{"missing-name", "missing-description"},
		},
		{
			name: "valid skill no warnings",
			skill: Skill{
				Path:        "/test/good.md",
				Content:     "---\nname: test\ndescription: foo\n---\nbody content here",
				Size:        50,
				Frontmatter: map[string]string{"name": "test", "description": "foo"},
			},
			wantTypes: nil,
		},
		{
			name: "nil frontmatter map",
			skill: Skill{
				Path:        "/test/nilmap.md",
				Content:     "some content without frontmatter",
				Size:        32,
				Frontmatter: nil,
			},
			wantTypes: []string{"missing-name", "missing-description"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := ValidateSkill(tt.skill)

			if tt.wantTypes == nil {
				if len(warnings) != 0 {
					types := make([]string, len(warnings))
					for i, w := range warnings {
						types[i] = w.Type
					}
					t.Errorf("expected no warnings, got %v", types)
				}
				return
			}

			gotTypes := make(map[string]bool)
			for _, w := range warnings {
				gotTypes[w.Type] = true
				// Every warning should carry the skill's path.
				if w.Path != tt.skill.Path {
					t.Errorf("warning path = %q, want %q", w.Path, tt.skill.Path)
				}
				// Every warning should have a non-empty message.
				if w.Message == "" {
					t.Errorf("warning type %q has empty message", w.Type)
				}
			}

			for _, wantType := range tt.wantTypes {
				if !gotTypes[wantType] {
					t.Errorf("expected warning type %q not found in %v", wantType, gotTypes)
				}
			}
		})
	}
}

// TestValidateSkill_EmptyFileShortCircuits verifies that an empty file
// returns only the "empty-file" warning and does not produce additional
// warnings for missing name/description.
func TestValidateSkill_EmptyFileShortCircuits(t *testing.T) {
	skill := Skill{
		Path:        "/test/empty.md",
		Content:     "",
		Size:        0,
		Frontmatter: map[string]string{},
	}

	warnings := ValidateSkill(skill)
	if len(warnings) != 1 {
		t.Fatalf("expected exactly 1 warning, got %d", len(warnings))
	}
	if warnings[0].Type != "empty-file" {
		t.Errorf("expected type 'empty-file', got %q", warnings[0].Type)
	}
}

// ---------------------------------------------------------------------------
// ValidateAll
// ---------------------------------------------------------------------------

func TestValidateAll(t *testing.T) {
	skills := []Skill{
		{
			Path:        "/test/good.md",
			Content:     "---\nname: good\ndescription: ok\n---\nbody",
			Size:        40,
			Frontmatter: map[string]string{"name": "good", "description": "ok"},
		},
		{
			Path:    "/test/empty.md",
			Content: "",
			Size:    0,
		},
		{
			Path:        "/test/noname.md",
			Content:     "---\ndescription: test\n---\nbody",
			Size:        30,
			Frontmatter: map[string]string{"description": "test"},
		},
	}

	result := ValidateAll(skills)

	// The valid skill should not appear in the result.
	if _, ok := result["/test/good.md"]; ok {
		t.Error("valid skill should not have warnings in ValidateAll result")
	}

	// The empty skill should be present.
	if warnings, ok := result["/test/empty.md"]; !ok {
		t.Error("expected /test/empty.md in result")
	} else if len(warnings) == 0 {
		t.Error("expected at least one warning for /test/empty.md")
	}

	// The noname skill should be present.
	if warnings, ok := result["/test/noname.md"]; !ok {
		t.Error("expected /test/noname.md in result")
	} else {
		hasNameWarning := false
		for _, w := range warnings {
			if w.Type == "missing-name" {
				hasNameWarning = true
			}
		}
		if !hasNameWarning {
			t.Error("expected 'missing-name' warning for /test/noname.md")
		}
	}
}

func TestValidateAll_AllValid(t *testing.T) {
	skills := []Skill{
		{
			Path:        "/test/a.md",
			Content:     "---\nname: a\ndescription: ok\n---\nbody",
			Size:        35,
			Frontmatter: map[string]string{"name": "a", "description": "ok"},
		},
	}

	result := ValidateAll(skills)
	if len(result) != 0 {
		t.Errorf("expected empty result for all-valid skills, got %d entries", len(result))
	}
}

func TestValidateAll_Empty(t *testing.T) {
	result := ValidateAll(nil)
	if len(result) != 0 {
		t.Errorf("expected empty result for nil input, got %d entries", len(result))
	}
}
