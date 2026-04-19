package skills

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// MarketplaceSource describes a GitHub repository that contains community-shared AI skills.
type MarketplaceSource struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Repo     string   `json:"repo"`     // e.g. "PatrickJS/awesome-cursorrules"
	Branch   string   `json:"branch"`   // e.g. "main"
	BasePath string   `json:"basePath"` // subdirectory containing the skills
	Tool     ToolType `json:"tool"`     // what tool these skills are for
}

// MarketplaceSkill represents a single skill file available in a marketplace source.
type MarketplaceSkill struct {
	Name        string `json:"name"`
	Path        string `json:"path"`        // path within the repo
	DownloadURL string `json:"downloadUrl"` // raw content URL
	Size        int64  `json:"size"`
	Source      string `json:"source"` // source ID
	Tool        string `json:"tool"`
}

// DefaultSources is the curated list of community skill repositories.
var DefaultSources = []MarketplaceSource{
	// Cursor
	{
		ID:       "awesome-cursorrules",
		Name:     "Awesome Cursor Rules",
		Repo:     "PatrickJS/awesome-cursorrules",
		Branch:   "main",
		BasePath: "rules",
		Tool:     ToolCursor,
	},
	{
		ID:       "awesome-cursor-rules-mdc",
		Name:     "Cursor Rules (MDC format)",
		Repo:     "sanjeed5/awesome-cursor-rules-mdc",
		Branch:   "main",
		BasePath: "rules-mdc",
		Tool:     ToolCursor,
	},
	// Multi-tool agents/skills
	{
		ID:       "ok-skills",
		Name:     "OK Skills (Multi-tool)",
		Repo:     "mxyhi/ok-skills",
		Branch:   "main",
		BasePath: "",
		Tool:     ToolCodex,
	},
	// Claude Code
	{
		ID:       "claude-code-system-prompts",
		Name:     "Claude Code Prompts",
		Repo:     "Piebald-AI/claude-code-system-prompts",
		Branch:   "main",
		BasePath: "",
		Tool:     ToolClaudeCode,
	},
	// Windsurf
	{
		ID:       "awesome-windsurf-rules",
		Name:     "Windsurf Rules",
		Repo:     "bklieger-groq/awesome-windsurf-rules",
		Branch:   "main",
		BasePath: "rules",
		Tool:     ToolWindsurf,
	},
	// Copilot
	{
		ID:       "awesome-copilot-instructions",
		Name:     "Copilot Instructions",
		Repo:     "alexcg1/awesome-copilot-instructions",
		Branch:   "main",
		BasePath: "instructions",
		Tool:     ToolCopilot,
	},
}

// --- In-memory cache for GitHub tree responses ---

type treeCache struct {
	mu      sync.RWMutex
	entries map[string]treeCacheEntry
}

type treeCacheEntry struct {
	skills    []MarketplaceSkill
	fetchedAt time.Time
}

var cache = &treeCache{
	entries: make(map[string]treeCacheEntry),
}

const cacheTTL = 10 * time.Minute

func (c *treeCache) get(key string) ([]MarketplaceSkill, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	if !ok || time.Since(entry.fetchedAt) > cacheTTL {
		return nil, false
	}
	return entry.skills, true
}

func (c *treeCache) set(key string, skills []MarketplaceSkill) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = treeCacheEntry{
		skills:    skills,
		fetchedAt: time.Now(),
	}
}

// --- GitHub API types ---

type githubTree struct {
	SHA       string           `json:"sha"`
	Tree      []githubTreeNode `json:"tree"`
	Truncated bool             `json:"truncated"`
}

type githubTreeNode struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
	Type string `json:"type"` // "blob" or "tree"
	Size int64  `json:"size"`
	SHA  string `json:"sha"`
	URL  string `json:"url"`
}

// FetchMarketplaceSkills retrieves the list of skill files from a GitHub source.
// It uses the Trees API (single request, recursive) and caches the result in memory.
func FetchMarketplaceSkills(source MarketplaceSource) ([]MarketplaceSkill, error) {
	cacheKey := source.ID

	// Check cache first
	if cached, ok := cache.get(cacheKey); ok {
		return cached, nil
	}

	// Parse owner/repo
	parts := strings.SplitN(source.Repo, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repo format: %s (expected owner/repo)", source.Repo)
	}
	owner, repo := parts[0], parts[1]

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1",
		owner, repo, source.Branch)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "AgentDesk-Skill-Manager")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch tree: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	var tree githubTree
	if err := json.NewDecoder(resp.Body).Decode(&tree); err != nil {
		return nil, fmt.Errorf("decode tree: %w", err)
	}

	basePath := strings.TrimSuffix(source.BasePath, "/")
	if basePath != "" {
		basePath += "/"
	}

	var skills []MarketplaceSkill
	for _, node := range tree.Tree {
		if node.Type != "blob" {
			continue
		}

		// Filter to files within the base path
		if basePath != "" && !strings.HasPrefix(node.Path, basePath) {
			continue
		}

		// Skip hidden files and READMEs
		filename := filepath.Base(node.Path)
		lower := strings.ToLower(filename)
		if strings.HasPrefix(filename, ".") || lower == "readme.md" {
			continue
		}

		// Build the raw download URL
		downloadURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s",
			owner, repo, source.Branch, node.Path)

		// Derive a display name from the path
		name := deriveSkillName(node.Path, basePath)

		skills = append(skills, MarketplaceSkill{
			Name:        name,
			Path:        node.Path,
			DownloadURL: downloadURL,
			Size:        node.Size,
			Source:      source.ID,
			Tool:        string(source.Tool),
		})
	}

	// Cache the result
	cache.set(cacheKey, skills)

	return skills, nil
}

// deriveSkillName extracts a human-readable name from a file path.
// For paths like "rules/next-js/.cursorrules", it returns "next-js".
// For flat files like "rules/my-skill.mdc", it returns "my-skill".
func deriveSkillName(path, basePath string) string {
	rel := strings.TrimPrefix(path, basePath)

	// If the file is inside a subdirectory, use the first subdirectory name
	parts := strings.Split(rel, "/")
	if len(parts) > 1 {
		return parts[0]
	}

	// Otherwise use filename without extension
	name := filepath.Base(rel)
	ext := filepath.Ext(name)
	if ext != "" {
		name = strings.TrimSuffix(name, ext)
	}
	return name
}

// FetchSkillContent downloads the raw content of a skill file from GitHub.
func FetchSkillContent(downloadURL string) (string, error) {
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "AgentDesk-Skill-Manager")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch content: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	return string(body), nil
}

// InstallSkill writes the given content to the appropriate local directory for the tool.
// Returns the full path of the installed file.
func InstallSkill(content string, tool ToolType, filename string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}

	// Determine install directory based on tool
	installDir := resolveInstallDir(home, tool)

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return "", fmt.Errorf("create directory %s: %w", installDir, err)
	}

	// Sanitize filename
	filename = sanitizeFilename(filename)

	fullPath := filepath.Join(installDir, filename)

	// Don't overwrite existing files
	if _, err := os.Stat(fullPath); err == nil {
		return "", fmt.Errorf("file already exists: %s", fullPath)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return fullPath, nil
}

// resolveInstallDir determines the correct local directory for installing a skill
// based on the target tool.
func resolveInstallDir(home string, tool ToolType) string {
	switch tool {
	case ToolClaudeCode:
		return filepath.Join(home, ".claude", "commands")
	case ToolGeminiCLI:
		return filepath.Join(home, ".gemini", "agents")
	case ToolCursor:
		return filepath.Join(home, ".cursor", "rules")
	case ToolKiro:
		return filepath.Join(home, ".kiro", "steering")
	case ToolWindsurf:
		// Windsurf uses project-level files; install to a global location
		return filepath.Join(home, ".windsurf", "rules")
	case ToolCopilot:
		return filepath.Join(home, ".github", "agents")
	case ToolAider:
		return filepath.Join(home, ".aider")
	case ToolAmp:
		return filepath.Join(home, ".amp", "skills")
	case ToolCodex:
		return filepath.Join(home, ".codex", "agents")
	default:
		return filepath.Join(home, ".agentdesk", "skills")
	}
}

// sanitizeFilename cleans up a filename, ensuring it has a valid extension
// and no path traversal characters.
func sanitizeFilename(name string) string {
	// Remove any directory components
	name = filepath.Base(name)

	// Replace spaces with hyphens, remove non-safe chars
	replacer := strings.NewReplacer(" ", "-")
	name = replacer.Replace(name)

	if name == "" || name == "." || name == ".." {
		name = "skill.md"
	}

	return name
}
