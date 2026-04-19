package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
)

// MCPServer represents a single MCP server entry from a tool's config.
type MCPServer struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Tool    string            `json:"tool"`
	Source  string            `json:"source"`
}

// MCPToolConfig groups MCP servers found for a specific AI tool.
type MCPToolConfig struct {
	Tool    string      `json:"tool"`
	Label   string      `json:"label"`
	Path    string      `json:"path"`
	Exists  bool        `json:"exists"`
	Servers []MCPServer `json:"servers"`
}

// mcpToolDef describes where to find MCP config for a given tool.
type mcpToolDef struct {
	Tool    string
	Label   string
	Path    string // resolved at runtime
	JSONKey string // top-level key in the JSON file
}

// mcpToolDefs returns the list of tool definitions with paths resolved
// for the current OS.
func mcpToolDefs() []mcpToolDef {
	home, _ := os.UserHomeDir()

	claudeDesktopPath := filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	if runtime.GOOS == "linux" {
		claudeDesktopPath = filepath.Join(home, ".config", "Claude", "claude_desktop_config.json")
	} else if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData != "" {
			claudeDesktopPath = filepath.Join(appData, "Claude", "claude_desktop_config.json")
		}
	}

	return []mcpToolDef{
		{
			Tool:    "claude-desktop",
			Label:   "Claude Desktop",
			Path:    claudeDesktopPath,
			JSONKey: "mcpServers",
		},
		{
			Tool:    "claude-code",
			Label:   "Claude Code",
			Path:    filepath.Join(home, ".claude.json"),
			JSONKey: "mcpServers",
		},
		{
			Tool:    "cursor",
			Label:   "Cursor",
			Path:    filepath.Join(home, ".cursor", "mcp.json"),
			JSONKey: "mcpServers",
		},
		{
			Tool:    "windsurf",
			Label:   "Windsurf",
			Path:    filepath.Join(home, ".codeium", "windsurf", "mcp_config.json"),
			JSONKey: "mcpServers",
		},
		{
			Tool:    "copilot",
			Label:   "VS Code / Copilot",
			Path:    filepath.Join(home, ".config", "github-copilot", "xcode", "mcp.json"),
			JSONKey: "servers",
		},
		{
			Tool:    "copilot-intellij",
			Label:   "IntelliJ / Copilot",
			Path:    filepath.Join(home, ".config", "github-copilot", "intellij", "mcp.json"),
			JSONKey: "servers",
		},
		{
			Tool:    "kiro",
			Label:   "Kiro",
			Path:    filepath.Join(home, ".kiro", "settings", "mcp.json"),
			JSONKey: "mcpServers",
		},
		{
			Tool:    "gemini-cli",
			Label:   "Gemini CLI",
			Path:    filepath.Join(home, ".gemini", "settings.json"),
			JSONKey: "mcpServers",
		},
		{
			Tool:    "amp",
			Label:   "Amp",
			Path:    filepath.Join(home, ".config", "amp", "settings.json"),
			JSONKey: "mcpServers",
		},
	}
}

// LoadAllMCPConfigs reads MCP config files from all supported AI tools
// and returns a unified list grouped by tool. Missing or malformed files
// are handled gracefully — the tool entry is still returned with an empty
// server list.
func LoadAllMCPConfigs() []MCPToolConfig {
	defs := mcpToolDefs()
	results := make([]MCPToolConfig, 0, len(defs))

	for _, def := range defs {
		cfg := MCPToolConfig{
			Tool:    def.Tool,
			Label:   def.Label,
			Path:    def.Path,
			Exists:  false,
			Servers: []MCPServer{},
		}

		data, err := os.ReadFile(def.Path)
		if err != nil {
			results = append(results, cfg)
			continue
		}
		cfg.Exists = true

		servers := parseMCPServers(data, def.JSONKey, def.Tool, def.Path)
		if servers != nil {
			cfg.Servers = servers
		}

		results = append(results, cfg)
	}

	return results
}

// parseMCPServers extracts MCP server entries from raw JSON data. It handles
// the common structure where servers are stored as a map under a top-level key.
func parseMCPServers(data []byte, jsonKey, tool, source string) []MCPServer {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	serversRaw, ok := raw[jsonKey]
	if !ok {
		return nil
	}

	var serverMap map[string]json.RawMessage
	if err := json.Unmarshal(serversRaw, &serverMap); err != nil {
		return nil
	}

	servers := make([]MCPServer, 0, len(serverMap))
	for name, entry := range serverMap {
		server := parseSingleServer(entry, name, tool, source)
		servers = append(servers, server)
	}

	// Sort by name for stable output
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Name < servers[j].Name
	})

	return servers
}

// parseSingleServer parses a single MCP server entry from raw JSON.
// It handles both stdio-based (command + args) and HTTP-based (url) configs.
func parseSingleServer(data json.RawMessage, name, tool, source string) MCPServer {
	var entry struct {
		Command string            `json:"command"`
		Args    []string          `json:"args"`
		Env     map[string]string `json:"env"`
		URL     string            `json:"url"`
		Type    string            `json:"type"`
	}
	_ = json.Unmarshal(data, &entry)

	// Mask env values — only expose key names, never actual secrets.
	maskedEnv := make(map[string]string, len(entry.Env))
	for k := range entry.Env {
		maskedEnv[k] = "\u2022\u2022\u2022"
	}

	server := MCPServer{
		Name:    name,
		Command: entry.Command,
		Args:    entry.Args,
		Env:     maskedEnv,
		URL:     entry.URL,
		Tool:    tool,
		Source:  source,
	}

	if server.Args == nil {
		server.Args = []string{}
	}

	return server
}
