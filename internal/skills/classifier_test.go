package skills

import (
	"testing"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name     string
		absPath  string
		wantTool ToolType
		wantCat  Category
	}{
		// Claude Code
		{
			name:     "claude code agent",
			absPath:  "/Users/test/.claude/agents/foo.md",
			wantTool: ToolClaudeCode,
			wantCat:  CatAgent,
		},
		{
			name:     "claude code command",
			absPath:  "/Users/test/.claude/commands/bar.md",
			wantTool: ToolClaudeCode,
			wantCat:  CatCommand,
		},
		{
			name:     "claude code plugin",
			absPath:  "/Users/test/.claude/plugins/baz.md",
			wantTool: ToolClaudeCode,
			wantCat:  CatSkill,
		},
		{
			name:     "claude.md config",
			absPath:  "/some/project/CLAUDE.md",
			wantTool: ToolClaudeCode,
			wantCat:  CatConfig,
		},
		// Cursor
		{
			name:     "cursorrules file",
			absPath:  "/some/project/.cursorrules",
			wantTool: ToolCursor,
			wantCat:  CatRule,
		},
		{
			name:     "cursorignore file",
			absPath:  "/some/project/.cursorignore",
			wantTool: ToolCursor,
			wantCat:  CatRule,
		},
		{
			name:     "cursor mdc rule",
			absPath:  "/Users/test/.cursor/rules/foo.mdc",
			wantTool: ToolCursor,
			wantCat:  CatRule,
		},
		// Kiro
		{
			name:     "kiro steering",
			absPath:  "/Users/test/.kiro/steering/foo.md",
			wantTool: ToolKiro,
			wantCat:  CatRule,
		},
		{
			name:     "kiro agent json",
			absPath:  "/Users/test/.kiro/agents/foo.json",
			wantTool: ToolKiro,
			wantCat:  CatAgent,
		},
		// Gemini
		{
			name:     "gemini.md config",
			absPath:  "/some/project/GEMINI.md",
			wantTool: ToolGeminiCLI,
			wantCat:  CatConfig,
		},
		{
			name:     "gemini agent",
			absPath:  "/Users/test/.gemini/agents/helper.md",
			wantTool: ToolGeminiCLI,
			wantCat:  CatAgent,
		},
		// Windsurf
		{
			name:     "windsurfrules file",
			absPath:  "/some/project/.windsurfrules",
			wantTool: ToolWindsurf,
			wantCat:  CatRule,
		},
		// Copilot
		{
			name:     "copilot instructions",
			absPath:  "/some/project/.github/copilot-instructions.md",
			wantTool: ToolCopilot,
			wantCat:  CatConfig,
		},
		{
			name:     "copilot agent",
			absPath:  "/some/project/.github/agents/helper.md",
			wantTool: ToolCopilot,
			wantCat:  CatAgent,
		},
		// Aider
		{
			name:     "aider config yml",
			absPath:  "/some/project/.aider.conf.yml",
			wantTool: ToolAider,
			wantCat:  CatConfig,
		},
		{
			name:     "aider conventions",
			absPath:  "/some/project/aider.conventions.md",
			wantTool: ToolAider,
			wantCat:  CatRule,
		},
		// Amp
		{
			name:     "amp skill",
			absPath:  "/Users/test/.amp/skills/foo.md",
			wantTool: ToolAmp,
			wantCat:  CatSkill,
		},
		// Codex
		{
			name:     "codex agents.md",
			absPath:  "/some/project/agents.md",
			wantTool: ToolCodex,
			wantCat:  CatConfig,
		},
		{
			name:     "codex agent toml",
			absPath:  "/Users/test/.codex/agents/helper.toml",
			wantTool: ToolCodex,
			wantCat:  CatAgent,
		},
		// Unknown
		{
			name:     "unknown random file",
			absPath:  "/some/project/random_file.txt",
			wantTool: ToolUnknown,
			wantCat:  CatConfig,
		},
		{
			name:     "unknown go file",
			absPath:  "/some/project/main.go",
			wantTool: ToolUnknown,
			wantCat:  CatConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Classify(tt.absPath)
			if result.Tool != tt.wantTool {
				t.Errorf("Classify(%q).Tool = %q, want %q", tt.absPath, result.Tool, tt.wantTool)
			}
			if result.Category != tt.wantCat {
				t.Errorf("Classify(%q).Category = %q, want %q", tt.absPath, result.Category, tt.wantCat)
			}
		})
	}
}

func TestClassify_CaseInsensitive(t *testing.T) {
	// The classifier lowercases filenames, so mixed-case paths should still match.
	tests := []struct {
		name     string
		absPath  string
		wantTool ToolType
	}{
		{
			name:     "uppercase CLAUDE.md",
			absPath:  "/project/CLAUDE.MD",
			wantTool: ToolClaudeCode,
		},
		{
			name:     "mixed case Gemini.md",
			absPath:  "/project/Gemini.Md",
			wantTool: ToolGeminiCLI,
		},
		{
			name:     "uppercase .WINDSURFRULES",
			absPath:  "/project/.WINDSURFRULES",
			wantTool: ToolWindsurf,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Classify(tt.absPath)
			if result.Tool != tt.wantTool {
				t.Errorf("Classify(%q).Tool = %q, want %q", tt.absPath, result.Tool, tt.wantTool)
			}
		})
	}
}
