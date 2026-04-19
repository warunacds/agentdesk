package team

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// TeamState tracks the team connection and managed files.
type TeamState struct {
	ServerURL    string                 `json:"server_url"`
	Token        string                 `json:"token"`
	OrgName      string                 `json:"org_name"`
	OrgID        string                 `json:"org_id"`
	Connected    bool                   `json:"connected"`
	LastSync     time.Time              `json:"last_sync"`
	ManagedFiles map[string]ManagedFile `json:"managed_files"`
}

// ManagedFile represents a file that is governed by the team.
type ManagedFile struct {
	Path    string `json:"path"`
	SHA256  string `json:"sha256"`
	Version int    `json:"version"`
	Tool    string `json:"tool"`
}

// stateFilePath returns the path to the team state file.
func stateFilePath(configDir string) string {
	return filepath.Join(configDir, "team-state.json")
}

// LoadState reads team state from disk. Returns empty defaults on error.
func LoadState(configDir string) TeamState {
	data, err := os.ReadFile(stateFilePath(configDir))
	if err != nil {
		return TeamState{ManagedFiles: map[string]ManagedFile{}}
	}
	var state TeamState
	if err := json.Unmarshal(data, &state); err != nil {
		return TeamState{ManagedFiles: map[string]ManagedFile{}}
	}
	if state.ManagedFiles == nil {
		state.ManagedFiles = map[string]ManagedFile{}
	}
	return state
}

// SaveState writes team state to disk.
func SaveState(configDir string, state TeamState) error {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(stateFilePath(configDir), data, 0600)
}

// ClearState removes the team state file.
func ClearState(configDir string) error {
	return os.Remove(stateFilePath(configDir))
}
