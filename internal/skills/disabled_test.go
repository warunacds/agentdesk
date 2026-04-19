package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDisabledStore_MissingFileReturnsEmpty(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	store := LoadDisabledStore()
	if len(store.Entries) != 0 {
		t.Fatalf("expected empty store, got %d entries", len(store.Entries))
	}
}

func TestSaveAndLoadDisabledStore(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	want := DisabledStore{
		Entries: []DisabledEntry{
			{OriginalPath: "/home/u/.claude/agents/foo.md", Tool: "claude-code", DisabledAt: 1234567890},
		},
	}
	if err := SaveDisabledStore(want); err != nil {
		t.Fatal(err)
	}

	got := LoadDisabledStore()
	if len(got.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got.Entries))
	}
	if got.Entries[0].OriginalPath != want.Entries[0].OriginalPath {
		t.Fatalf("path mismatch: got %s want %s", got.Entries[0].OriginalPath, want.Entries[0].OriginalPath)
	}

	if _, err := os.Stat(filepath.Join(tmp, ".agentdesk", "disabled-skills.json")); err != nil {
		t.Fatalf("expected disabled-skills.json: %v", err)
	}
}

func TestDisableSkill_MovesFileAndRecordsEntry(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	toolDir := filepath.Join(tmp, ".claude", "agents")
	if err := os.MkdirAll(toolDir, 0755); err != nil {
		t.Fatal(err)
	}
	skillPath := filepath.Join(toolDir, "my-skill.md")
	if err := os.WriteFile(skillPath, []byte("# Hello"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := DisableSkill(skillPath, ToolClaudeCode); err != nil {
		t.Fatalf("DisableSkill failed: %v", err)
	}

	if _, err := os.Stat(skillPath); !os.IsNotExist(err) {
		t.Fatal("original file should be removed")
	}

	disabledPath := filepath.Join(tmp, ".agentdesk", "disabled", "claude-code", "my-skill.md")
	if _, err := os.Stat(disabledPath); err != nil {
		t.Fatalf("disabled file should exist: %v", err)
	}

	store := LoadDisabledStore()
	if len(store.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(store.Entries))
	}
	if store.Entries[0].OriginalPath != skillPath {
		t.Fatalf("wrong path: got %s", store.Entries[0].OriginalPath)
	}
}

func TestEnableSkill_RestoresFileAndRemovesEntry(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	toolDir := filepath.Join(tmp, ".claude", "agents")
	os.MkdirAll(toolDir, 0755)
	skillPath := filepath.Join(toolDir, "my-skill.md")
	os.WriteFile(skillPath, []byte("# Hello"), 0644)

	DisableSkill(skillPath, ToolClaudeCode)

	if err := EnableSkill(skillPath); err != nil {
		t.Fatalf("EnableSkill failed: %v", err)
	}

	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("original file should be restored: %v", err)
	}
	if string(data) != "# Hello" {
		t.Fatalf("content changed: got %s", string(data))
	}

	store := LoadDisabledStore()
	if len(store.Entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(store.Entries))
	}
}

func TestIsDisabled(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	toolDir := filepath.Join(tmp, ".claude", "agents")
	os.MkdirAll(toolDir, 0755)
	skillPath := filepath.Join(toolDir, "my-skill.md")
	os.WriteFile(skillPath, []byte("x"), 0644)

	if IsDisabled(skillPath) {
		t.Fatal("fresh skill should not be disabled")
	}

	DisableSkill(skillPath, ToolClaudeCode)

	if !IsDisabled(skillPath) {
		t.Fatal("skill should be disabled after DisableSkill")
	}
}

func TestListDisabledSkills_ReturnsEntriesWithMetadata(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	toolDir := filepath.Join(tmp, ".claude", "agents")
	os.MkdirAll(toolDir, 0755)
	skillPath := filepath.Join(toolDir, "x.md")
	os.WriteFile(skillPath, []byte("---\nname: X Skill\n---\n# X"), 0644)

	DisableSkill(skillPath, ToolClaudeCode)

	list := ListDisabledSkills()
	if len(list) != 1 {
		t.Fatalf("expected 1, got %d", len(list))
	}
	if list[0].OriginalPath != skillPath {
		t.Fatalf("wrong path: %s", list[0].OriginalPath)
	}
	if list[0].Name == "" {
		t.Fatal("expected Name populated from frontmatter")
	}
	if list[0].Name != "X Skill" {
		t.Fatalf("expected Name='X Skill', got %q", list[0].Name)
	}
	if list[0].Size == 0 {
		t.Fatal("expected non-zero Size")
	}
}

func TestListDisabledSkills_EmptyWhenNone(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	list := ListDisabledSkills()
	if len(list) != 0 {
		t.Fatalf("expected empty, got %d", len(list))
	}
}

func TestListDisabledSkills_FallsBackToFilename(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	toolDir := filepath.Join(tmp, ".claude", "agents")
	os.MkdirAll(toolDir, 0755)
	skillPath := filepath.Join(toolDir, "no-frontmatter.md")
	os.WriteFile(skillPath, []byte("# Just a title"), 0644)

	DisableSkill(skillPath, ToolClaudeCode)

	list := ListDisabledSkills()
	if len(list) != 1 {
		t.Fatalf("expected 1, got %d", len(list))
	}
	if list[0].Name != "no-frontmatter" {
		t.Fatalf("expected filename stem 'no-frontmatter', got %q", list[0].Name)
	}
}
