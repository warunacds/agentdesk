package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Project is a user-pinned project directory used to scope skill scanning.
type Project struct {
	ID      string `json:"id"`
	Path    string `json:"path"`
	Name    string `json:"name"`
	AddedAt int64  `json:"addedAt"`
}

// ProjectStore is the top-level JSON structure persisted to disk
// for pinned projects.
type ProjectStore struct {
	Projects []Project `json:"projects"`
}

// ProjectOverrides maps a project path to a list of skill file paths
// that should be disabled when scoped to that project.
type ProjectOverrides struct {
	DisabledSkills map[string][]string `json:"disabledSkills"`
}

// projectStorePath returns ~/.agentdesk/projects.json.
func projectStorePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agentdesk", "projects.json")
}

// projectOverridesPath returns ~/.agentdesk/project-overrides.json.
func projectOverridesPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agentdesk", "project-overrides.json")
}

// LoadProjectStore reads the project store from disk.
// Returns an empty store if the file does not exist or is unreadable.
func LoadProjectStore() ProjectStore {
	data, err := os.ReadFile(projectStorePath())
	if err != nil {
		return ProjectStore{Projects: []Project{}}
	}
	var s ProjectStore
	if err := json.Unmarshal(data, &s); err != nil {
		return ProjectStore{Projects: []Project{}}
	}
	if s.Projects == nil {
		s.Projects = []Project{}
	}
	return s
}

// SaveProjectStore writes the project store to disk, creating
// the ~/.agentdesk/ directory if it does not exist.
func SaveProjectStore(s ProjectStore) error {
	p := projectStorePath()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

// LoadProjectOverrides reads the project overrides from disk.
// Returns empty overrides if the file does not exist or is unreadable.
func LoadProjectOverrides() ProjectOverrides {
	data, err := os.ReadFile(projectOverridesPath())
	if err != nil {
		return ProjectOverrides{DisabledSkills: map[string][]string{}}
	}
	var o ProjectOverrides
	if err := json.Unmarshal(data, &o); err != nil {
		return ProjectOverrides{DisabledSkills: map[string][]string{}}
	}
	if o.DisabledSkills == nil {
		o.DisabledSkills = map[string][]string{}
	}
	return o
}

// SaveProjectOverrides writes the project overrides to disk, creating
// the ~/.agentdesk/ directory if it does not exist.
func SaveProjectOverrides(o ProjectOverrides) error {
	p := projectOverridesPath()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

// AddProject pins a project directory under ~/.agentdesk/projects.json.
// If name is empty, the directory's base name is used.
// Returns an error if the path is already pinned.
func AddProject(path string, name string) (Project, error) {
	store := LoadProjectStore()
	for _, p := range store.Projects {
		if p.Path == path {
			return Project{}, fmt.Errorf("project already pinned: %s", path)
		}
	}
	if name == "" {
		name = filepath.Base(path)
	}
	p := Project{
		ID:      uuid.New().String(),
		Path:    path,
		Name:    name,
		AddedAt: time.Now().Unix(),
	}
	store.Projects = append(store.Projects, p)
	if err := SaveProjectStore(store); err != nil {
		return Project{}, err
	}
	return p, nil
}

// RemoveProject unpins a project by ID.
// Returns nil if the project does not exist.
func RemoveProject(id string) error {
	store := LoadProjectStore()
	idx := -1
	for i, p := range store.Projects {
		if p.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil
	}
	store.Projects = append(store.Projects[:idx], store.Projects[idx+1:]...)
	return SaveProjectStore(store)
}

// RenameProject updates the display name of a pinned project.
// Returns nil if the project does not exist.
func RenameProject(id string, name string) error {
	store := LoadProjectStore()
	for i := range store.Projects {
		if store.Projects[i].ID == id {
			store.Projects[i].Name = name
			return SaveProjectStore(store)
		}
	}
	return nil
}

// SetProjectOverride toggles whether a skill is disabled for the given
// project. Adding an already-disabled skill or removing a not-disabled
// skill is a no-op.
func SetProjectOverride(projectPath string, skillPath string, disabled bool) error {
	o := LoadProjectOverrides()
	list := o.DisabledSkills[projectPath]
	present := false
	for _, s := range list {
		if s == skillPath {
			present = true
			break
		}
	}
	if disabled && !present {
		list = append(list, skillPath)
	} else if !disabled && present {
		out := list[:0]
		for _, s := range list {
			if s != skillPath {
				out = append(out, s)
			}
		}
		list = out
	}
	if len(list) == 0 {
		delete(o.DisabledSkills, projectPath)
	} else {
		o.DisabledSkills[projectPath] = list
	}
	return SaveProjectOverrides(o)
}

// GetProjectOverrides returns the list of disabled skill paths for a project.
// Returns an empty slice if there are no overrides.
func GetProjectOverrides(projectPath string) []string {
	o := LoadProjectOverrides()
	list := o.DisabledSkills[projectPath]
	if list == nil {
		return []string{}
	}
	return list
}

// ScanProjectSkills scans for skills inside the given project directory only,
// filtering out any globally-scoped results from the user's home directories.
func ScanProjectSkills(projectPath string) []Skill {
	all := Scan([]string{projectPath})
	out := make([]Skill, 0, len(all))
	prefix := projectPath
	if !strings.HasSuffix(prefix, string(os.PathSeparator)) {
		prefix = prefix + string(os.PathSeparator)
	}
	for _, s := range all {
		if strings.HasPrefix(s.Path, prefix) {
			out = append(out, s)
		}
	}
	return out
}
