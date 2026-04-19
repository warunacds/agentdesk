package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PluginInfo represents a single installed plugin for an AI tool.
type PluginInfo struct {
	Name        string `json:"name"`
	Marketplace string `json:"marketplace"`
	Version     string `json:"version"`
	Scope       string `json:"scope"`
	InstallPath string `json:"installPath"`
	InstalledAt string `json:"installedAt"`
	LastUpdated string `json:"lastUpdated"`
}

// PluginToolConfig groups plugins found for a specific AI tool.
type PluginToolConfig struct {
	Tool      string       `json:"tool"`
	Label     string       `json:"label"`
	Available bool         `json:"available"`
	Plugins   []PluginInfo `json:"plugins"`
}

// pluginToolDef describes where to find plugin data for a given tool.
type pluginToolDef struct {
	Tool      string
	Label     string
	Dir       string // directory to check for plugin support
	Manifest  string // path to the manifest file (empty if none)
	Available bool   // whether this tool has a known plugin system
}

// pluginToolDefs returns the list of plugin tool definitions with paths resolved
// for the current user.
func pluginToolDefs() []pluginToolDef {
	home, _ := os.UserHomeDir()

	return []pluginToolDef{
		{
			Tool:      "claude-code",
			Label:     "Claude Code",
			Dir:       filepath.Join(home, ".claude", "plugins"),
			Manifest:  filepath.Join(home, ".claude", "plugins", "installed_plugins.json"),
			Available: true,
		},
		{
			Tool:      "codex",
			Label:     "Codex",
			Dir:       filepath.Join(home, ".codex", "plugins"),
			Manifest:  "",
			Available: false,
		},
		{
			Tool:      "gemini-cli",
			Label:     "Gemini CLI",
			Dir:       filepath.Join(home, ".gemini", "plugins"),
			Manifest:  "",
			Available: false,
		},
		{
			Tool:      "opencode",
			Label:     "OpenCode",
			Dir:       filepath.Join(home, ".opencode", "plugins"),
			Manifest:  "",
			Available: false,
		},
	}
}

// LoadAllPlugins reads plugin manifests from all supported AI tools and
// returns a unified list grouped by tool. Missing or malformed files are
// handled gracefully -- the tool entry is still returned with an empty
// plugin list.
func LoadAllPlugins() []PluginToolConfig {
	defs := pluginToolDefs()
	results := make([]PluginToolConfig, 0, len(defs))

	for _, def := range defs {
		cfg := PluginToolConfig{
			Tool:      def.Tool,
			Label:     def.Label,
			Available: def.Available,
			Plugins:   []PluginInfo{},
		}

		// If the tool has no known plugin system, check if the directory
		// appeared (in case they add support later). If it exists with a
		// manifest, try to parse it.
		if !def.Available {
			if _, err := os.Stat(def.Dir); err == nil {
				// Directory exists -- check for a manifest
				candidate := filepath.Join(def.Dir, "installed_plugins.json")
				if plugins := parseClaudePluginManifest(candidate); len(plugins) > 0 {
					cfg.Available = true
					cfg.Plugins = plugins
				}
			}
			results = append(results, cfg)
			continue
		}

		if def.Manifest == "" {
			results = append(results, cfg)
			continue
		}

		plugins := parseClaudePluginManifest(def.Manifest)
		if plugins != nil {
			cfg.Plugins = plugins
		}

		results = append(results, cfg)
	}

	return results
}

// claudePluginManifest represents the structure of Claude Code's
// installed_plugins.json file.
type claudePluginManifest struct {
	Version int                              `json:"version"`
	Plugins map[string][]claudePluginEntry   `json:"plugins"`
}

type claudePluginEntry struct {
	Scope       string `json:"scope"`
	InstallPath string `json:"installPath"`
	Version     string `json:"version"`
	InstalledAt string `json:"installedAt"`
	LastUpdated string `json:"lastUpdated"`
}

// parseClaudePluginManifest reads and parses an installed_plugins.json file,
// returning a sorted list of PluginInfo entries. Returns nil on any error.
func parseClaudePluginManifest(path string) []PluginInfo {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var manifest claudePluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil
	}

	var plugins []PluginInfo
	for key, entries := range manifest.Plugins {
		name, marketplace := splitPluginKey(key)
		for _, entry := range entries {
			plugins = append(plugins, PluginInfo{
				Name:        name,
				Marketplace: marketplace,
				Version:     entry.Version,
				Scope:       entry.Scope,
				InstallPath: entry.InstallPath,
				InstalledAt: entry.InstalledAt,
				LastUpdated: entry.LastUpdated,
			})
		}
	}

	// Sort by name for stable output
	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Name < plugins[j].Name
	})

	return plugins
}

// splitPluginKey splits a plugin key of the form "name@marketplace" into its
// two parts. If there is no "@", the whole key is treated as the name and
// marketplace is empty.
func splitPluginKey(key string) (name, marketplace string) {
	idx := strings.LastIndex(key, "@")
	if idx < 0 {
		return key, ""
	}
	return key[:idx], key[idx+1:]
}
