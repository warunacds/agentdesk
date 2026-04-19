package sync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// SyncManifest tracks which file paths are opted-in for synchronisation.
// When Mode is "all", every syncable file is included regardless of the
// SyncedPaths list. When Mode is "selective", only files whose relative
// paths appear in SyncedPaths are synced.
type SyncManifest struct {
	Mode        string   `json:"mode"`        // "selective" or "all"
	SyncedPaths []string `json:"syncedPaths"` // tilde-relative paths opted-in
}

// manifestPath returns the path to the persistent sync-manifest file.
func manifestPath() string {
	return filepath.Join(dataDir(), "sync-manifest.json")
}

// LoadManifest reads the sync manifest from ~/.agentdesk/sync-manifest.json.
// If the file does not exist or cannot be parsed, a default manifest with
// mode "selective" and an empty path list is returned so callers always
// receive a usable value.
func LoadManifest() *SyncManifest {
	data, err := os.ReadFile(manifestPath())
	if err != nil {
		return &SyncManifest{
			Mode:        "selective",
			SyncedPaths: []string{},
		}
	}
	var m SyncManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return &SyncManifest{
			Mode:        "selective",
			SyncedPaths: []string{},
		}
	}
	if m.SyncedPaths == nil {
		m.SyncedPaths = []string{}
	}
	if m.Mode == "" {
		m.Mode = "selective"
	}
	return &m
}

// SaveManifest writes the manifest to ~/.agentdesk/sync-manifest.json, creating
// the directory if it does not already exist.
func SaveManifest(m *SyncManifest) error {
	if err := os.MkdirAll(dataDir(), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(manifestPath(), data, 0644)
}

// IsPathSynced reports whether relPath is opted-in for sync. If the
// manifest mode is "all", every path is considered synced. Otherwise the
// path must appear in the SyncedPaths list.
func (m *SyncManifest) IsPathSynced(relPath string) bool {
	if m.Mode == "all" {
		return true
	}
	for _, p := range m.SyncedPaths {
		if p == relPath {
			return true
		}
	}
	return false
}

// AddPath adds relPath to the synced paths list if it is not already
// present. The list is kept sorted for deterministic serialisation.
func (m *SyncManifest) AddPath(relPath string) {
	for _, p := range m.SyncedPaths {
		if p == relPath {
			return // already present
		}
	}
	m.SyncedPaths = append(m.SyncedPaths, relPath)
	sort.Strings(m.SyncedPaths)
}

// RemovePath removes relPath from the synced paths list. It is a no-op
// if the path is not present.
func (m *SyncManifest) RemovePath(relPath string) {
	filtered := m.SyncedPaths[:0]
	for _, p := range m.SyncedPaths {
		if p != relPath {
			filtered = append(filtered, p)
		}
	}
	m.SyncedPaths = filtered
}

// AddPaths adds multiple paths to the synced paths list, deduplicating
// against existing entries.
func (m *SyncManifest) AddPaths(relPaths []string) {
	for _, rp := range relPaths {
		m.AddPath(rp)
	}
}

// GetPaths returns a copy of all synced paths. The returned slice is
// independent of the manifest's internal state.
func (m *SyncManifest) GetPaths() []string {
	out := make([]string, len(m.SyncedPaths))
	copy(out, m.SyncedPaths)
	return out
}
