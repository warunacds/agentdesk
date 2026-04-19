package sync

import (
	"os"
	"path/filepath"
	"strings"
)

// FileInfo holds the content hash and filesystem metadata for a single scanned file.
type FileInfo struct {
	Hash     string
	Modified int64
	Size     int64
}

// SyncDirs returns the list of directories whose contents should be
// synchronised between paired peers. Each entry is an absolute path
// derived from the current user's home directory.
func SyncDirs() []string {
	home, _ := os.UserHomeDir()
	return []string{
		filepath.Join(home, ".claude", "agents"),
		filepath.Join(home, ".claude", "commands"),
		filepath.Join(home, ".gemini", "agents"),
		filepath.Join(home, ".kiro", "steering"),
		filepath.Join(home, ".kiro", "agents"),
		filepath.Join(home, ".amp", "skills"),
		filepath.Join(home, ".cursor", "rules"),
		filepath.Join(home, ".codex", "agents"),
		filepath.Join(home, ".agentdesk"),
	}
}

// SyncFiles returns individual files (not directories) that should be
// synchronised between paired peers. These are MCP configuration files
// that live outside the standard watched directories.
func SyncFiles() []string {
	home, _ := os.UserHomeDir()
	return []string{
		filepath.Join(home, ".claude.json"),
		filepath.Join(home, ".cursor", "mcp.json"),
		filepath.Join(home, ".gemini", "settings.json"),
		filepath.Join(home, ".config", "amp", "settings.json"),
	}
}

// IsSyncable reports whether absPath falls within one of the watched
// sync directories or is one of the individually synced files.
func IsSyncable(absPath string) bool {
	for _, dir := range SyncDirs() {
		if strings.HasPrefix(absPath, dir) {
			return true
		}
	}
	for _, f := range SyncFiles() {
		if absPath == f {
			return true
		}
	}
	return false
}

// SkipSyncFile reports whether relPath names a file that must never be
// synced (identity keys, peer lists, and the sync state file itself).
func SkipSyncFile(relPath string) bool {
	base := filepath.Base(relPath)
	skip := map[string]bool{
		"identity.key":       true,
		"peers.json":         true,
		"sync-state.json":    true,
		"sync-manifest.json": true,
	}
	return skip[base]
}

// ScanSyncableFiles walks every watched directory and returns a map of
// relative paths to their FileInfo. Directories that do not exist are
// silently skipped; individual files that cannot be hashed are also
// skipped so a single unreadable file does not block the whole scan.
func ScanSyncableFiles() (map[string]FileInfo, error) {
	files := make(map[string]FileInfo)
	for _, dir := range SyncDirs() {
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			rel := toRelPath(path)
			if SkipSyncFile(rel) {
				return nil
			}
			hash, err := HashFile(path)
			if err != nil {
				return nil
			}
			info, _ := d.Info()
			files[rel] = FileInfo{
				Hash:     hash,
				Modified: info.ModTime().Unix(),
				Size:     info.Size(),
			}
			return nil
		})
	}
	// Also scan individually watched files (MCP configs)
	for _, f := range SyncFiles() {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		rel := toRelPath(f)
		if SkipSyncFile(rel) {
			continue
		}
		hash, err := HashFile(f)
		if err != nil {
			continue
		}
		files[rel] = FileInfo{
			Hash:     hash,
			Modified: info.ModTime().Unix(),
			Size:     info.Size(),
		}
	}

	return files, nil
}

// toRelPath converts an absolute filesystem path to a tilde-relative
// path (e.g. "~/.<tool>/agents/foo.md"). If the path does not start
// with the home directory it is returned unchanged.
func toRelPath(absPath string) string {
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(absPath, home) {
		return "~" + absPath[len(home):]
	}
	return absPath
}

// resolveAbsPath converts a tilde-relative path back to an absolute
// filesystem path. Non-tilde paths are joined with the home directory
// as a fallback.
func resolveAbsPath(relPath string) string {
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(relPath, "~/") {
		return filepath.Join(home, relPath[2:])
	}
	return filepath.Join(home, relPath)
}
