package team

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TeamSync is the engine that periodically pulls files from the team server.
type TeamSync struct {
	client    *Client
	configDir string
	homeDir   string
	lastSync  time.Time
	interval  time.Duration
	stopCh    chan struct{}
	running   bool
}

// NewTeamSync creates a new team sync engine.
func NewTeamSync(client *Client, configDir string) *TeamSync {
	home, _ := os.UserHomeDir()
	return &TeamSync{
		client:    client,
		configDir: configDir,
		homeDir:   home,
		interval:  5 * time.Minute,
		stopCh:    make(chan struct{}),
	}
}

// Start begins periodic syncing in a goroutine.
func (ts *TeamSync) Start() {
	if ts.running {
		return
	}
	ts.running = true

	// Load last sync time from state
	state := LoadState(ts.configDir)
	ts.lastSync = state.LastSync

	// Do an initial sync immediately
	go func() {
		if err := ts.SyncNow(); err != nil {
			log.Printf("team-sync: initial sync error: %v", err)
		}
	}()

	// Start periodic sync
	go func() {
		ticker := time.NewTicker(ts.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := ts.SyncNow(); err != nil {
					log.Printf("team-sync: periodic sync error: %v", err)
				}
			case <-ts.stopCh:
				return
			}
		}
	}()
}

// Stop halts the periodic sync.
func (ts *TeamSync) Stop() {
	if !ts.running {
		return
	}
	ts.running = false
	close(ts.stopCh)
}

// SyncNow performs an immediate sync cycle.
func (ts *TeamSync) SyncNow() error {
	state := LoadState(ts.configDir)

	// Determine the "since" time — use epoch if never synced
	since := ts.lastSync
	if since.IsZero() {
		since = time.Unix(0, 0)
	}

	changes, err := ts.client.GetChanges(since)
	if err != nil {
		return err
	}

	for _, change := range changes {
		localPath, err := ts.resolveLocalPath(change.Path)
		if err != nil {
			log.Printf("team-sync: skipping %s: %v", change.Path, err)
			continue
		}

		if change.Deleted {
			// Remove deleted file
			if _, exists := state.ManagedFiles[change.Path]; exists {
				if err := os.Remove(localPath); err != nil && !os.IsNotExist(err) {
					log.Printf("team-sync: failed to remove %s: %v", localPath, err)
				}
				delete(state.ManagedFiles, change.Path)
				log.Printf("team-sync: removed %s", change.Path)
			}
			continue
		}

		// Pull file content
		fc, err := ts.client.PullFile(change.Path)
		if err != nil {
			log.Printf("team-sync: failed to pull %s: %v", change.Path, err)
			continue
		}

		// Write to local filesystem
		if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
			log.Printf("team-sync: failed to create dir for %s: %v", localPath, err)
			continue
		}
		if err := os.WriteFile(localPath, []byte(fc.Content), 0644); err != nil {
			log.Printf("team-sync: failed to write %s: %v", localPath, err)
			continue
		}

		// Track in managed files
		state.ManagedFiles[change.Path] = ManagedFile{
			Path:    change.Path,
			SHA256:  change.SHA256,
			Version: change.Version,
			Tool:    change.Tool,
		}
		log.Printf("team-sync: synced %s (v%d)", change.Path, change.Version)
	}

	// Update last sync time
	now := time.Now().UTC()
	ts.lastSync = now
	state.LastSync = now

	if err := SaveState(ts.configDir, state); err != nil {
		log.Printf("team-sync: failed to save state: %v", err)
		return err
	}

	return nil
}

// resolveLocalPath converts a relative API path to an absolute local path.
// Paths from the API are relative to the user's home directory
// (e.g. ".claude/agents/code-reviewer.md" -> "~/.claude/agents/code-reviewer.md").
// It validates that the resolved path stays within homeDir to prevent path traversal.
func (ts *TeamSync) resolveLocalPath(apiPath string) (string, error) {
	abs := filepath.Join(ts.homeDir, apiPath)
	// Ensure the resolved path is within the home directory
	if !strings.HasPrefix(abs, ts.homeDir+string(filepath.Separator)) && abs != ts.homeDir {
		return "", fmt.Errorf("path traversal rejected: %q resolves outside home", apiPath)
	}
	return abs, nil
}
