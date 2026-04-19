package sync

import (
	"testing"
)

// ---------------------------------------------------------------------------
// IsPathSynced
// ---------------------------------------------------------------------------

func TestIsPathSynced(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		paths   []string
		query   string
		want    bool
	}{
		{
			name:  "selective mode path present",
			mode:  "selective",
			paths: []string{"~/.claude/agents/foo.md"},
			query: "~/.claude/agents/foo.md",
			want:  true,
		},
		{
			name:  "selective mode path absent",
			mode:  "selective",
			paths: []string{"~/.claude/agents/foo.md"},
			query: "~/.claude/agents/bar.md",
			want:  false,
		},
		{
			name:  "selective mode empty list",
			mode:  "selective",
			paths: []string{},
			query: "~/.claude/agents/foo.md",
			want:  false,
		},
		{
			name:  "all mode syncs everything",
			mode:  "all",
			paths: []string{},
			query: "anything-at-all",
			want:  true,
		},
		{
			name:  "all mode with populated list",
			mode:  "all",
			paths: []string{"~/.claude/agents/foo.md"},
			query: "~/.claude/agents/bar.md",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &SyncManifest{
				Mode:        tt.mode,
				SyncedPaths: tt.paths,
			}
			got := m.IsPathSynced(tt.query)
			if got != tt.want {
				t.Errorf("IsPathSynced(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AddPath
// ---------------------------------------------------------------------------

func TestAddPath(t *testing.T) {
	m := &SyncManifest{Mode: "selective", SyncedPaths: []string{}}

	m.AddPath("~/.claude/agents/foo.md")
	if !m.IsPathSynced("~/.claude/agents/foo.md") {
		t.Error("path should be synced after AddPath")
	}

	// Adding the same path again should not duplicate it.
	m.AddPath("~/.claude/agents/foo.md")
	count := 0
	for _, p := range m.SyncedPaths {
		if p == "~/.claude/agents/foo.md" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected path to appear once, got %d times", count)
	}
}

func TestAddPath_MaintainsSortOrder(t *testing.T) {
	m := &SyncManifest{Mode: "selective", SyncedPaths: []string{}}

	m.AddPath("c-path")
	m.AddPath("a-path")
	m.AddPath("b-path")

	if len(m.SyncedPaths) != 3 {
		t.Fatalf("expected 3 paths, got %d", len(m.SyncedPaths))
	}

	for i := 1; i < len(m.SyncedPaths); i++ {
		if m.SyncedPaths[i] < m.SyncedPaths[i-1] {
			t.Errorf("paths not sorted: %q comes after %q", m.SyncedPaths[i], m.SyncedPaths[i-1])
		}
	}
}

// ---------------------------------------------------------------------------
// RemovePath
// ---------------------------------------------------------------------------

func TestRemovePath(t *testing.T) {
	m := &SyncManifest{
		Mode:        "selective",
		SyncedPaths: []string{"~/.claude/agents/bar.md", "~/.claude/agents/foo.md"},
	}

	m.RemovePath("~/.claude/agents/foo.md")
	if m.IsPathSynced("~/.claude/agents/foo.md") {
		t.Error("path should not be synced after RemovePath")
	}

	// The other path should still be present.
	if !m.IsPathSynced("~/.claude/agents/bar.md") {
		t.Error("unrelated path should still be synced after RemovePath of a different path")
	}
}

func TestRemovePath_Nonexistent(t *testing.T) {
	m := &SyncManifest{
		Mode:        "selective",
		SyncedPaths: []string{"~/.claude/agents/foo.md"},
	}

	// Removing a path that does not exist should be a no-op.
	m.RemovePath("~/.claude/agents/not-here.md")
	if len(m.SyncedPaths) != 1 {
		t.Errorf("expected 1 path after removing nonexistent, got %d", len(m.SyncedPaths))
	}
}

// ---------------------------------------------------------------------------
// AddPaths
// ---------------------------------------------------------------------------

func TestAddPaths(t *testing.T) {
	m := &SyncManifest{Mode: "selective", SyncedPaths: []string{}}

	m.AddPaths([]string{"b-path", "a-path", "c-path"})

	if len(m.SyncedPaths) != 3 {
		t.Fatalf("expected 3 paths, got %d", len(m.SyncedPaths))
	}

	// Should be sorted.
	for i := 1; i < len(m.SyncedPaths); i++ {
		if m.SyncedPaths[i] < m.SyncedPaths[i-1] {
			t.Errorf("paths not sorted after AddPaths")
		}
	}
}

func TestAddPaths_DeduplicatesExisting(t *testing.T) {
	m := &SyncManifest{
		Mode:        "selective",
		SyncedPaths: []string{"a-path"},
	}

	m.AddPaths([]string{"a-path", "b-path"})

	if len(m.SyncedPaths) != 2 {
		t.Errorf("expected 2 paths after dedup, got %d", len(m.SyncedPaths))
	}
}

// ---------------------------------------------------------------------------
// GetPaths
// ---------------------------------------------------------------------------

func TestGetPaths_ReturnsCopy(t *testing.T) {
	m := &SyncManifest{
		Mode:        "selective",
		SyncedPaths: []string{"a", "b", "c"},
	}

	paths := m.GetPaths()
	if len(paths) != 3 {
		t.Fatalf("expected 3 paths, got %d", len(paths))
	}

	// Mutating the returned slice should not affect the manifest.
	paths[0] = "mutated"
	if m.SyncedPaths[0] == "mutated" {
		t.Error("GetPaths did not return an independent copy")
	}
}

func TestGetPaths_EmptyManifest(t *testing.T) {
	m := &SyncManifest{
		Mode:        "selective",
		SyncedPaths: []string{},
	}

	paths := m.GetPaths()
	if len(paths) != 0 {
		t.Errorf("expected 0 paths, got %d", len(paths))
	}
}

// ---------------------------------------------------------------------------
// Mode switching
// ---------------------------------------------------------------------------

func TestModeSwitch(t *testing.T) {
	m := &SyncManifest{
		Mode:        "selective",
		SyncedPaths: []string{"a"},
	}

	// In selective mode, only "a" is synced.
	if m.IsPathSynced("b") {
		t.Error("selective: 'b' should not be synced")
	}

	// Switch to "all" mode.
	m.Mode = "all"
	if !m.IsPathSynced("b") {
		t.Error("all: 'b' should be synced")
	}
	if !m.IsPathSynced("anything") {
		t.Error("all: any path should be synced")
	}

	// Switch back.
	m.Mode = "selective"
	if m.IsPathSynced("b") {
		t.Error("selective after switch: 'b' should not be synced")
	}
	if !m.IsPathSynced("a") {
		t.Error("selective after switch: 'a' should still be synced")
	}
}
