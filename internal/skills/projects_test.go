package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectStore_EmptyOnMissingFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	store := LoadProjectStore()
	if len(store.Projects) != 0 {
		t.Fatalf("expected empty, got %d", len(store.Projects))
	}
}

func TestProjectStore_SaveAndLoad(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	want := ProjectStore{
		Projects: []Project{
			{ID: "a1", Path: "/home/u/app", Name: "app", AddedAt: 111},
		},
	}
	if err := SaveProjectStore(want); err != nil {
		t.Fatal(err)
	}
	got := LoadProjectStore()
	if len(got.Projects) != 1 || got.Projects[0].Path != "/home/u/app" {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(tmp, ".agentdesk", "projects.json")); err != nil {
		t.Fatal(err)
	}
}

func TestProjectOverrides_EmptyOnMissingFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	o := LoadProjectOverrides()
	if len(o.DisabledSkills) != 0 {
		t.Fatal("expected empty map")
	}
}

func TestProjectOverrides_SaveAndLoad(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	want := ProjectOverrides{
		DisabledSkills: map[string][]string{
			"/home/u/app": {"/home/u/.claude/skills/x/SKILL.md"},
		},
	}
	if err := SaveProjectOverrides(want); err != nil {
		t.Fatal(err)
	}
	got := LoadProjectOverrides()
	if len(got.DisabledSkills["/home/u/app"]) != 1 {
		t.Fatalf("mismatch: %+v", got)
	}
}

func TestAddProject(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	p, err := AddProject("/home/u/app", "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Path != "/home/u/app" {
		t.Fatalf("path mismatch: %s", p.Path)
	}
	if p.Name != "app" {
		t.Fatalf("expected name=app, got %s", p.Name)
	}
	if p.ID == "" {
		t.Fatal("expected ID generated")
	}
	if p.AddedAt == 0 {
		t.Fatal("expected AddedAt set")
	}

	got := LoadProjectStore()
	if len(got.Projects) != 1 {
		t.Fatalf("expected 1, got %d", len(got.Projects))
	}
}

func TestAddProject_CustomName(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	p, _ := AddProject("/home/u/app", "My App")
	if p.Name != "My App" {
		t.Fatalf("expected custom name, got %s", p.Name)
	}
}

func TestAddProject_Duplicate(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	AddProject("/home/u/app", "")
	_, err := AddProject("/home/u/app", "")
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestRemoveProject(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	p, _ := AddProject("/home/u/app", "")
	if err := RemoveProject(p.ID); err != nil {
		t.Fatal(err)
	}
	got := LoadProjectStore()
	if len(got.Projects) != 0 {
		t.Fatal("expected 0 projects after remove")
	}
}

func TestRenameProject(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	p, _ := AddProject("/home/u/app", "")
	if err := RenameProject(p.ID, "Renamed"); err != nil {
		t.Fatal(err)
	}
	got := LoadProjectStore()
	if got.Projects[0].Name != "Renamed" {
		t.Fatalf("rename failed: %s", got.Projects[0].Name)
	}
}

func TestSetProjectOverride_Add(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	if err := SetProjectOverride("/home/u/app", "/g/SKILL.md", true); err != nil {
		t.Fatal(err)
	}
	o := LoadProjectOverrides()
	if len(o.DisabledSkills["/home/u/app"]) != 1 {
		t.Fatalf("expected 1, got %+v", o)
	}
}

func TestSetProjectOverride_Remove(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	SetProjectOverride("/home/u/app", "/g/SKILL.md", true)
	SetProjectOverride("/home/u/app", "/g/SKILL.md", false)
	o := LoadProjectOverrides()
	if len(o.DisabledSkills["/home/u/app"]) != 0 {
		t.Fatalf("expected empty, got %+v", o)
	}
}

func TestSetProjectOverride_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	SetProjectOverride("/home/u/app", "/g/SKILL.md", true)
	SetProjectOverride("/home/u/app", "/g/SKILL.md", true)
	o := LoadProjectOverrides()
	if len(o.DisabledSkills["/home/u/app"]) != 1 {
		t.Fatalf("duplicate added: %+v", o)
	}
}

func TestScanProjectSkills(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	projectDir := filepath.Join(tmp, "code", "my-app")
	claudeDir := filepath.Join(projectDir, ".claude", "agents")
	os.MkdirAll(claudeDir, 0755)
	os.WriteFile(filepath.Join(claudeDir, "project-agent.md"), []byte("# Local"), 0644)

	globalDir := filepath.Join(tmp, ".claude", "agents")
	os.MkdirAll(globalDir, 0755)
	os.WriteFile(filepath.Join(globalDir, "global-agent.md"), []byte("# Global"), 0644)

	result := ScanProjectSkills(projectDir)
	if len(result) != 1 {
		t.Fatalf("expected 1 local skill, got %d", len(result))
	}
	if filepath.Base(result[0].Path) != "project-agent.md" {
		t.Fatalf("wrong skill: %s", result[0].Path)
	}
}
