package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

// FileVersion records the known state of a single file, including its content
// hash, modification timestamp, size, and a vector clock tracking per-peer
// write counts. Deleted files retain their vector clock so that deletions
// propagate correctly during conflict resolution.
type FileVersion struct {
	Hash      string            `json:"hash"`
	Modified  int64             `json:"modified"`
	Size      int64             `json:"size"`
	Versions  map[string]uint64 `json:"versions"`
	Deleted   bool              `json:"deleted,omitempty"`
	DeletedAt int64             `json:"deletedAt,omitempty"`
}

// SyncConflict records a detected concurrent edit that requires manual
// resolution. Both the local and remote content are captured so the user
// can compare them in the UI.
type SyncConflict struct {
	Path          string `json:"path"`
	LocalHash     string `json:"localHash"`
	RemoteHash    string `json:"remoteHash"`
	RemotePeer    string `json:"remotePeer"`
	DetectedAt    int64  `json:"detectedAt"`
	LocalContent  string `json:"localContent"`
	RemoteContent string `json:"remoteContent"`
}

// SyncState is the on-disk representation of everything this node knows about
// the synchronised file set. Files is keyed by the vault-relative path.
// Conflicts holds any unresolved concurrent edits that need user intervention.
type SyncState struct {
	Files     map[string]*FileVersion `json:"files"`
	LocalID   string                  `json:"localId"`
	Conflicts []SyncConflict          `json:"conflicts,omitempty"`
}

// syncStatePath returns the path to the persistent sync-state file.
func syncStatePath() string {
	return filepath.Join(dataDir(), "sync-state.json")
}

// LoadSyncState reads the sync state from disk and returns it. If the file
// does not exist or cannot be parsed, a fresh empty state is returned so the
// caller always receives a usable value.
func LoadSyncState() *SyncState {
	data, err := os.ReadFile(syncStatePath())
	if err != nil {
		return &SyncState{Files: make(map[string]*FileVersion)}
	}
	var state SyncState
	if err := json.Unmarshal(data, &state); err != nil {
		return &SyncState{Files: make(map[string]*FileVersion)}
	}
	if state.Files == nil {
		state.Files = make(map[string]*FileVersion)
	}
	return &state
}

// Save persists the SyncState to disk as indented JSON. It creates the data
// directory if it does not already exist.
func (s *SyncState) Save() error {
	if err := os.MkdirAll(dataDir(), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(syncStatePath(), data, 0644)
}

// RecordLocalChange updates the tracked state for relPath to reflect a locally
// authored write. It increments the local node's vector-clock component so
// remote peers can detect the change.
func (s *SyncState) RecordLocalChange(relPath string, hash string, modified, size int64) {
	fv, ok := s.Files[relPath]
	if !ok {
		fv = &FileVersion{Versions: map[string]uint64{}}
		s.Files[relPath] = fv
	}
	fv.Hash = hash
	fv.Modified = modified
	fv.Size = size
	fv.Deleted = false
	fv.Versions[s.LocalID]++
}

// RecordRemoteChange stores the FileVersion received from a remote peer for
// relPath, replacing any previously held version entirely.
func (s *SyncState) RecordRemoteChange(relPath string, fv FileVersion) {
	s.Files[relPath] = &fv
}

// RecordDeletion marks relPath as deleted at the given Unix timestamp and
// increments the local vector-clock component so the deletion propagates.
func (s *SyncState) RecordDeletion(relPath string, timestamp int64) {
	fv, ok := s.Files[relPath]
	if !ok {
		fv = &FileVersion{Versions: map[string]uint64{}}
		s.Files[relPath] = fv
	}
	fv.Deleted = true
	fv.DeletedAt = timestamp
	fv.Versions[s.LocalID]++
}

// HashFile computes the SHA-256 digest of the file at path and returns it as a
// lowercase hex string.
func HashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}
