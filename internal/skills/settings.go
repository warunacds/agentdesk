package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings stores user preferences and custom scan paths.
type Settings struct {
	CustomPaths       map[string][]string `json:"customPaths"`
	Theme             string              `json:"theme"`
	ScanOnStartup     bool                `json:"scanOnStartup"`
	RunInBackground   bool                `json:"runInBackground"`
	LastUpdateCheck   int64               `json:"lastUpdateCheck"`
	DismissedVersion  string              `json:"dismissedVersion"`
}

// DefaultSettings returns a Settings with sensible defaults.
func DefaultSettings() Settings {
	return Settings{
		CustomPaths:   map[string][]string{},
		Theme:         "dark",
		ScanOnStartup: true,
	}
}

// settingsFilePath returns ~/.agentdesk/settings.json.
func settingsFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agentdesk", "settings.json")
}

// LoadSettings reads settings from disk.
// Returns defaults if the file does not exist or is unreadable.
func LoadSettings() Settings {
	data, err := os.ReadFile(settingsFilePath())
	if err != nil {
		return DefaultSettings()
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return DefaultSettings()
	}
	// Ensure non-nil map even if JSON had null
	if s.CustomPaths == nil {
		s.CustomPaths = map[string][]string{}
	}
	// Apply defaults for zero-value fields
	if s.Theme == "" {
		s.Theme = "dark"
	}
	return s
}

// SaveSettings writes settings to disk, creating ~/.agentdesk/ if needed.
func SaveSettings(s Settings) error {
	fp := settingsFilePath()
	if err := os.MkdirAll(filepath.Dir(fp), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(fp, data, 0644)
}

// DefaultScanPaths returns the built-in scan roots for reference display.
func DefaultScanPaths() map[string][]string {
	home, _ := os.UserHomeDir()
	return map[string][]string{
		string(ToolClaudeCode): {
			filepath.Join(home, ".claude", "agents"),
			filepath.Join(home, ".claude", "commands"),
		},
		string(ToolGeminiCLI): {
			filepath.Join(home, ".gemini", "agents"),
		},
		string(ToolKiro): {
			filepath.Join(home, ".kiro", "steering"),
			filepath.Join(home, ".kiro", "agents"),
		},
		string(ToolAmp): {
			filepath.Join(home, ".amp", "skills"),
		},
		string(ToolCursor): {
			filepath.Join(home, ".cursor", "rules"),
		},
		string(ToolCodex): {
			filepath.Join(home, ".codex", "agents"),
		},
	}
}
