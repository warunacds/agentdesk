package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const maxRecentFiles = 15

// RecentFile represents a recently opened or edited skill file.
type RecentFile struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Tool     string `json:"tool"`
	OpenedAt int64  `json:"openedAt"`
}

// RecentStore holds the list of recently accessed files.
type RecentStore struct {
	Files []RecentFile `json:"files"`
}

// recentFilePath returns ~/.agentdesk/recent.json.
func recentFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agentdesk", "recent.json")
}

// LoadRecent reads the recent store from disk.
// Returns an empty store if the file does not exist or is unreadable.
func LoadRecent() RecentStore {
	data, err := os.ReadFile(recentFilePath())
	if err != nil {
		return RecentStore{Files: []RecentFile{}}
	}
	var store RecentStore
	if err := json.Unmarshal(data, &store); err != nil {
		return RecentStore{Files: []RecentFile{}}
	}
	if store.Files == nil {
		store.Files = []RecentFile{}
	}
	return store
}

// SaveRecent writes the recent store to disk, creating ~/.agentdesk/ if needed.
func SaveRecent(store RecentStore) error {
	fp := recentFilePath()
	if err := os.MkdirAll(filepath.Dir(fp), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(fp, data, 0644)
}

// AddRecent adds a file to the front of the recent list, deduplicates by path,
// and caps the list at 15 entries.
func AddRecent(path, name, tool string) {
	store := LoadRecent()

	entry := RecentFile{
		Path:     path,
		Name:     name,
		Tool:     tool,
		OpenedAt: time.Now().Unix(),
	}

	// Remove any existing entry with the same path
	filtered := []RecentFile{entry}
	for _, f := range store.Files {
		if f.Path != path {
			filtered = append(filtered, f)
		}
	}

	// Cap at max
	if len(filtered) > maxRecentFiles {
		filtered = filtered[:maxRecentFiles]
	}

	store.Files = filtered
	_ = SaveRecent(store)
}

// ClearRecent removes all recent file entries.
func ClearRecent() error {
	return SaveRecent(RecentStore{Files: []RecentFile{}})
}
