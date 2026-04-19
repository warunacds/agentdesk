package skills

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

var candidateFilenames = map[string]bool{
	"claude.md": true, "gemini.md": true,
	".cursorrules": true, ".cursorignore": true,
	".windsurfrules": true, "copilot-instructions.md": true,
	".aider.conf.yml": true, "aider.conventions.md": true,
	"agents.md": true,
}

var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true,
	".next": true, "dist": true, "build": true, "__pycache__": true,
	"Library": true, "Applications": true, "Pictures": true,
	"Music": true, "Movies": true, ".Trash": true,
	".agentdesk": true,
}

func Scan(extraDirs []string) []Skill {
	home, _ := os.UserHomeDir()

	seen := map[string]bool{}
	var skills []Skill

	// Only scan specific known subdirectories — avoids permission prompts
	// and prevents picking up caches, plans, projects, etc.
	knownRoots := []string{
		// Claude Code
		filepath.Join(home, ".claude", "agents"),
		filepath.Join(home, ".claude", "skills"),
		filepath.Join(home, ".claude", "commands"),
		// Gemini CLI
		filepath.Join(home, ".gemini", "agents"),
		// Kiro
		filepath.Join(home, ".kiro", "steering"),
		filepath.Join(home, ".kiro", "agents"),
		// Amp
		filepath.Join(home, ".amp", "skills"),
		// Cursor
		filepath.Join(home, ".cursor", "rules"),
		// Codex/OpenAI
		filepath.Join(home, ".codex", "agents"),
	}

	// Pick up top-level config files and settings files directly
	settingsFiles := []string{
		// Claude Code
		filepath.Join(home, ".claude", "CLAUDE.md"),
		filepath.Join(home, ".claude", "settings.json"),
		filepath.Join(home, ".claude", "settings.local.json"),
		filepath.Join(home, ".claude.json"),
		// Gemini CLI
		filepath.Join(home, ".gemini", "GEMINI.md"),
		filepath.Join(home, ".gemini", "settings.json"),
		// Cursor
		filepath.Join(home, ".cursor", "mcp.json"),
		// Kiro
		filepath.Join(home, ".kiro", "settings", "mcp.json"),
		// Codex
		filepath.Join(home, ".codex", "config.toml"),
		// Amp
		filepath.Join(home, ".amp", "settings.json"),
		filepath.Join(home, ".config", "amp", "settings.json"),
	}
	for _, configPath := range settingsFiles {
		if fi, err := os.Stat(configPath); err == nil && !fi.IsDir() {
			collectFileByPath(configPath, seen, &skills)
		}
	}
	for _, root := range knownRoots {
		scanDeep(root, 8, seen, &skills)
	}

	// Scan user-added extra directories
	for _, root := range extraDirs {
		scanDeep(root, 8, seen, &skills)
	}

	// Deduplicate by real path: merge skills that resolve to the same file
	// (e.g. symlinked across multiple tool directories).
	skills = deduplicateByRealPath(skills)

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Modified > skills[j].Modified
	})
	return skills
}

func scanDeep(root string, maxDepth int, seen map[string]bool, skills *[]Skill) {
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return
	}

	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			rel, _ := filepath.Rel(root, path)
			if strings.Count(rel, string(os.PathSeparator)) > maxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		collectFile(path, d, seen, skills)
		return nil
	})
}

func collectFile(path string, d os.DirEntry, seen map[string]bool, skills *[]Skill) {
	filename := strings.ToLower(d.Name())
	pl := strings.ToLower(path)

	isCandidate := candidateFilenames[filename] ||
		(strings.Contains(pl, "/.claude/skills/") && filename == "skill.md") ||
		(strings.Contains(pl, "/.claude/agents/") && strings.HasSuffix(filename, ".md")) ||
		(strings.Contains(pl, "/.claude/commands/") && strings.HasSuffix(filename, ".md")) ||
		(strings.Contains(pl, "/.claude/plugins/") && strings.HasSuffix(filename, ".md")) ||
		(strings.Contains(pl, "/.gemini/agents/") && strings.HasSuffix(filename, ".md")) ||
		(strings.Contains(pl, "/.kiro/steering/") && strings.HasSuffix(filename, ".md")) ||
		(strings.Contains(pl, "/.kiro/agents/") && strings.HasSuffix(filename, ".json")) ||
		(strings.Contains(pl, "/.cursor/rules/") && strings.HasSuffix(filename, ".mdc")) ||
		(strings.Contains(pl, "/.github/agents/") && strings.HasSuffix(filename, ".md")) ||
		(strings.Contains(pl, "/.amp/skills/") && strings.HasSuffix(filename, ".md")) ||
		(strings.Contains(pl, "/.codex/agents/") && strings.HasSuffix(filename, ".toml"))

	if !isCandidate {
		return
	}

	// Skip README files — they're not actual skill/agent definitions
	if filename == "readme.md" {
		return
	}

	// Resolve symlinks to get the real path for deduplication
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		realPath = path
	}

	// Skip if this exact path was already visited (avoids re-scanning
	// the same directory entry). Real-path deduplication for symlinks
	// happens in Scan() as a post-processing merge pass.
	if seen[path] {
		return
	}

	result := Classify(path)
	if result.Tool == ToolUnknown {
		return
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return
	}

	fi, err := d.Info()
	if err != nil {
		return
	}

	fm, _ := ParseFrontmatter(string(content))
	name := fm["name"]
	if name == "" {
		name = strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
	}

	directory := ""
	var auxFiles []AuxFile
	if d.Name() == "SKILL.md" {
		directory = filepath.Dir(path)
		auxFiles = collectAuxFiles(directory)
	}

	seen[path] = true
	*skills = append(*skills, Skill{
		ID:             uuid.New().String(),
		Name:           name,
		Path:           path,
		RealPath:       realPath,
		Tool:           result.Tool,
		Tools:          []ToolType{result.Tool},
		Category:       result.Category,
		Content:        string(content),
		Frontmatter:    fm,
		Modified:       fi.ModTime().Unix(),
		Size:           fi.Size(),
		Directory:      directory,
		AuxiliaryFiles: auxFiles,
	})
}

func collectFileByPath(path string, seen map[string]bool, skills *[]Skill) {
	if seen[path] {
		return
	}

	// Resolve symlinks to get the real path for deduplication
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		realPath = path
	}

	fi, err := os.Stat(path)
	if err != nil {
		return
	}

	result := Classify(path)
	if result.Tool == ToolUnknown {
		return
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return
	}

	fm, _ := ParseFrontmatter(string(content))
	name := fm["name"]
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	seen[path] = true
	*skills = append(*skills, Skill{
		ID:          uuid.New().String(),
		Name:        name,
		Path:        path,
		RealPath:    realPath,
		Tool:        result.Tool,
		Tools:       []ToolType{result.Tool},
		Category:    result.Category,
		Content:     string(content),
		Frontmatter: fm,
		Modified:    fi.ModTime().Unix(),
		Size:        fi.Size(),
	})
}

// deduplicateByRealPath merges skills that resolve to the same real file path.
// For each group sharing a RealPath, the first skill's fields are kept as primary
// and all distinct Tool values are merged into its Tools slice.
func deduplicateByRealPath(skills []Skill) []Skill {
	// Map from real path to the index in the deduped slice
	realPathIndex := map[string]int{}
	var deduped []Skill

	for _, s := range skills {
		if idx, ok := realPathIndex[s.RealPath]; ok {
			// Merge: add this skill's tool to the existing entry's Tools list
			found := false
			for _, t := range deduped[idx].Tools {
				if t == s.Tool {
					found = true
					break
				}
			}
			if !found {
				deduped[idx].Tools = append(deduped[idx].Tools, s.Tool)
			}
		} else {
			realPathIndex[s.RealPath] = len(deduped)
			deduped = append(deduped, s)
		}
	}

	return deduped
}

func classifyAuxFile(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".markdown", ".mdc", ".mdx":
		return "markdown"
	case ".sh", ".bash", ".zsh", ".fish":
		return "script"
	case ".py", ".js", ".ts", ".rb", ".go", ".rs", ".json", ".yaml", ".yml", ".toml":
		return "code"
	default:
		return "other"
	}
}

func collectAuxFiles(dir string) []AuxFile {
	var out []AuxFile
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if skipDirs[name] {
				return filepath.SkipDir
			}
			return nil
		}
		base := d.Name()
		if base == "SKILL.md" || base == ".DS_Store" || base == "Thumbs.db" {
			return nil
		}
		if strings.ToLower(base) == "readme.md" {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		if strings.Count(rel, string(os.PathSeparator)) > 8 {
			return nil
		}
		out = append(out, AuxFile{
			Path:    path,
			RelPath: rel,
			Type:    classifyAuxFile(path),
		})
		return nil
	})
	return out
}

func ParseFrontmatter(content string) (map[string]string, string) {
	fm := map[string]string{}
	if !strings.HasPrefix(content, "---") {
		return fm, content
	}
	rest := content[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return fm, content
	}
	for _, line := range strings.Split(rest[:idx], "\n") {
		if k, v, ok := strings.Cut(line, ":"); ok {
			fm[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return fm, strings.TrimSpace(rest[idx+4:])
}

// ReplaceFrontmatter takes file content and a new frontmatter map, returns
// content with updated frontmatter. If the file has no existing frontmatter,
// it prepends a new block. Preserves existing key order where possible and
// appends new keys at the end.
func ReplaceFrontmatter(content string, newFM map[string]string) string {
	if len(newFM) == 0 {
		return content
	}

	var body string
	var existingOrder []string

	if strings.HasPrefix(content, "---") {
		rest := content[3:]
		idx := strings.Index(rest, "\n---")
		if idx >= 0 {
			// Parse existing key order
			for _, line := range strings.Split(rest[:idx], "\n") {
				if k, _, ok := strings.Cut(line, ":"); ok {
					key := strings.TrimSpace(k)
					if key != "" {
						existingOrder = append(existingOrder, key)
					}
				}
			}
			// rest[idx+4:] skips past "\n---" (4 chars).
			// The next char is the newline terminating the "---" line,
			// which we skip so the body starts cleanly after the closing fence.
			body = rest[idx+4:]
			if strings.HasPrefix(body, "\n") {
				body = body[1:]
			}
		} else {
			body = content
		}
	} else {
		body = content
	}

	// Build ordered key list: existing keys first (in order), then new keys
	seen := map[string]bool{}
	var orderedKeys []string
	for _, k := range existingOrder {
		if _, exists := newFM[k]; exists {
			orderedKeys = append(orderedKeys, k)
			seen[k] = true
		}
	}
	// Collect new keys not in existing order
	var newKeys []string
	for k := range newFM {
		if !seen[k] {
			newKeys = append(newKeys, k)
		}
	}
	sort.Strings(newKeys)
	orderedKeys = append(orderedKeys, newKeys...)

	// Build frontmatter block
	var sb strings.Builder
	sb.WriteString("---\n")
	for _, k := range orderedKeys {
		v := newFM[k]
		// Quote values that contain colons or special YAML chars
		if strings.ContainsAny(v, ":#{}[]|>&*!%@`") || strings.HasPrefix(v, "'") || strings.HasPrefix(v, "\"") {
			sb.WriteString(k + ": \"" + strings.ReplaceAll(v, "\"", "\\\"") + "\"\n")
		} else {
			sb.WriteString(k + ": " + v + "\n")
		}
	}
	sb.WriteString("---\n")
	sb.WriteString(body)

	return sb.String()
}

func GenerateTemplate(tool ToolType, name string) string {
	switch tool {
	case ToolClaudeCode:
		return "---\nname: " + name + "\ndescription: Describe this skill\n---\n\n# " + name + "\n\n"
	case ToolGeminiCLI:
		return "# " + name + "\n\n## General Instructions\n\n- \n\n## Coding Style\n\n- \n"
	case ToolKiro:
		return "---\nname: " + name + "\ndescription: Kiro steering file\ninclusion: auto\n---\n\n# " + name + "\n\n"
	case ToolCursor:
		return "# " + name + "\n\n## Guidelines\n\n- Rule 1\n- Rule 2\n"
	case ToolWindsurf:
		return "# " + name + "\n\n## Guidelines\n\n- Rule 1\n- Rule 2\n"
	case ToolCopilot:
		return "# Copilot Instructions\n\n## " + name + "\n\n"
	case ToolCodex:
		return "# Agents\n\n## " + name + "\n\n"
	default:
		return "# " + name + "\n\n"
	}
}

func DefaultFilename(tool ToolType) string {
	switch tool {
	case ToolClaudeCode:
		return "CLAUDE.md"
	case ToolGeminiCLI:
		return "GEMINI.md"
	case ToolKiro:
		return "steering.md"
	case ToolCursor:
		return ".cursorrules"
	case ToolWindsurf:
		return ".windsurfrules"
	case ToolCopilot:
		return "copilot-instructions.md"
	case ToolAider:
		return "aider.conventions.md"
	case ToolAmp:
		return "skill.md"
	case ToolCodex:
		return "AGENTS.md"
	default:
		return "skill.md"
	}
}

func DefaultSubdir(tool ToolType) string {
	switch tool {
	case ToolKiro:
		return ".kiro/steering"
	case ToolCopilot:
		return ".github"
	case ToolAmp:
		return ".amp/skills"
	case ToolCursor:
		return ".cursor/rules"
	default:
		return ""
	}
}

func NowUnix() int64 { return time.Now().Unix() }
