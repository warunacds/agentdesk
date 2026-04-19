package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// DisabledEntry records one skill file that has been disabled for a tool.
type DisabledEntry struct {
	OriginalPath string   `json:"originalPath"`
	Tool         ToolType `json:"tool"`
	DisabledAt   int64    `json:"disabledAt"`
}

// DisabledStore is persisted to ~/.agentdesk/disabled-skills.json.
type DisabledStore struct {
	Entries []DisabledEntry `json:"entries"`
}

func disabledStorePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agentdesk", "disabled-skills.json")
}

// disabledDir returns ~/.agentdesk/disabled/<tool>/.
func disabledDir(tool ToolType) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agentdesk", "disabled", string(tool))
}

// LoadDisabledStore reads the disabled-skills file. Returns empty store on error.
func LoadDisabledStore() DisabledStore {
	data, err := os.ReadFile(disabledStorePath())
	if err != nil {
		return DisabledStore{Entries: []DisabledEntry{}}
	}
	var s DisabledStore
	if err := json.Unmarshal(data, &s); err != nil {
		return DisabledStore{Entries: []DisabledEntry{}}
	}
	if s.Entries == nil {
		s.Entries = []DisabledEntry{}
	}
	return s
}

// SaveDisabledStore writes the store to disk, creating the directory if needed.
func SaveDisabledStore(s DisabledStore) error {
	p := disabledStorePath()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

// DisableSkill moves the file at path to ~/.agentdesk/disabled/<tool>/<filename>
// and records the original path. If already disabled, returns nil (idempotent).
func DisableSkill(path string, tool ToolType) error {
	store := LoadDisabledStore()
	for _, e := range store.Entries {
		if e.OriginalPath == path {
			return nil
		}
	}

	dir := disabledDir(tool)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	dest := filepath.Join(dir, filepath.Base(path))

	if _, err := os.Stat(dest); err == nil {
		dest = filepath.Join(dir, timestampedName(filepath.Base(path)))
	}

	if err := os.Rename(path, dest); err != nil {
		return err
	}

	store.Entries = append(store.Entries, DisabledEntry{
		OriginalPath: path,
		Tool:         tool,
		DisabledAt:   time.Now().Unix(),
	})
	return SaveDisabledStore(store)
}

// EnableSkill restores a previously disabled skill to its original path.
// If the skill is not in the store, returns nil (idempotent).
func EnableSkill(originalPath string) error {
	store := LoadDisabledStore()
	idx := -1
	var entry DisabledEntry
	for i, e := range store.Entries {
		if e.OriginalPath == originalPath {
			idx = i
			entry = e
			break
		}
	}
	if idx < 0 {
		return nil
	}

	src := filepath.Join(disabledDir(entry.Tool), filepath.Base(originalPath))
	if _, err := os.Stat(src); err != nil {
		store.Entries = append(store.Entries[:idx], store.Entries[idx+1:]...)
		return SaveDisabledStore(store)
	}

	if err := os.MkdirAll(filepath.Dir(originalPath), 0755); err != nil {
		return err
	}
	if err := os.Rename(src, originalPath); err != nil {
		return err
	}

	store.Entries = append(store.Entries[:idx], store.Entries[idx+1:]...)
	return SaveDisabledStore(store)
}

// IsDisabled returns true if the given original path is in the disabled store.
func IsDisabled(originalPath string) bool {
	store := LoadDisabledStore()
	for _, e := range store.Entries {
		if e.OriginalPath == originalPath {
			return true
		}
	}
	return false
}

func timestampedName(base string) string {
	return time.Now().Format("20060102-150405") + "-" + base
}

// DisabledSkillInfo is a disabled entry enriched with display metadata.
type DisabledSkillInfo struct {
	OriginalPath string   `json:"originalPath"`
	Tool         ToolType `json:"tool"`
	DisabledAt   int64    `json:"disabledAt"`
	Name         string   `json:"name"`
	Size         int64    `json:"size"`
}

// ListDisabledSkills returns the current store enriched with Name/Size from the
// disabled file on disk (best effort; missing files still included with empty fields).
func ListDisabledSkills() []DisabledSkillInfo {
	store := LoadDisabledStore()
	out := make([]DisabledSkillInfo, 0, len(store.Entries))
	for _, e := range store.Entries {
		info := DisabledSkillInfo{
			OriginalPath: e.OriginalPath,
			Tool:         e.Tool,
			DisabledAt:   e.DisabledAt,
		}
		disabledPath := filepath.Join(disabledDir(e.Tool), filepath.Base(e.OriginalPath))
		if st, err := os.Stat(disabledPath); err == nil {
			info.Size = st.Size()
		}
		if data, err := os.ReadFile(disabledPath); err == nil {
			fm, _ := ParseFrontmatter(string(data))
			if n := fm["name"]; n != "" {
				info.Name = n
			}
		}
		if info.Name == "" {
			base := filepath.Base(e.OriginalPath)
			info.Name = stripExt(base)
		}
		out = append(out, info)
	}
	return out
}

func stripExt(s string) string {
	ext := filepath.Ext(s)
	if ext != "" {
		return s[:len(s)-len(ext)]
	}
	return s
}
