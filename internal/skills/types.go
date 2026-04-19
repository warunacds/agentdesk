package skills

type ToolType string

const (
	ToolClaudeCode ToolType = "claude-code"
	ToolGeminiCLI  ToolType = "gemini-cli"
	ToolCursor     ToolType = "cursor"
	ToolKiro       ToolType = "kiro"
	ToolWindsurf   ToolType = "windsurf"
	ToolCopilot    ToolType = "copilot"
	ToolAider      ToolType = "aider"
	ToolAmp        ToolType = "amp"
	ToolCodex      ToolType = "codex"
	ToolUnknown    ToolType = "unknown"
)

type ToolMeta struct {
	ID    ToolType `json:"id"`
	Label string   `json:"label"`
	Color string   `json:"color"`
	Icon  string   `json:"icon"`
}

var KnownTools = []ToolMeta{
	{ToolClaudeCode, "Claude Code", "#d4772c", "\u2b21"},
	{ToolGeminiCLI, "Gemini CLI", "#4285f4", "\u25c7"},
	{ToolCursor, "Cursor", "#5c8af0", "\u25c8"},
	{ToolKiro, "Kiro", "#ff9900", "\u25ed"},
	{ToolWindsurf, "Windsurf", "#3ecf8e", "\u25ce"},
	{ToolCopilot, "Copilot", "#f0c040", "\u25c9"},
	{ToolAider, "Aider", "#c084fc", "\u25c6"},
	{ToolAmp, "Amp", "#f97316", "\u25b2"},
	{ToolCodex, "Codex", "#38bdf8", "\u25d0"},
}

type Category string

const (
	CatAgent   Category = "agent"
	CatSkill   Category = "skill"
	CatCommand Category = "command"
	CatRule    Category = "rule"
	CatConfig  Category = "config"
)

type Skill struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Path           string            `json:"path"`
	RealPath       string            `json:"realPath"`
	Tool           ToolType          `json:"tool"`
	Tools          []ToolType        `json:"tools"`
	Category       Category          `json:"category"`
	Content        string            `json:"content"`
	Frontmatter    map[string]string `json:"frontmatter"`
	Modified       int64             `json:"modified"`
	Size           int64             `json:"size"`
	Directory      string            `json:"directory"`
	AuxiliaryFiles []AuxFile         `json:"auxiliaryFiles"`
}

// AuxFile is a file inside a folder-based skill's directory,
// not the main SKILL.md itself.
type AuxFile struct {
	Path    string `json:"path"`
	RelPath string `json:"relPath"`
	Type    string `json:"type"` // "markdown" | "script" | "code" | "other"
}
