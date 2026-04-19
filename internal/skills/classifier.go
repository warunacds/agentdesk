package skills

import (
	"path/filepath"
	"strings"
)

type ClassifyResult struct {
	Tool     ToolType
	Category Category
}

func Classify(absPath string) ClassifyResult {
	filename := strings.ToLower(filepath.Base(absPath))
	p := strings.ToLower(absPath)

	switch {
	// Claude Code
	case filename == "claude.md":
		return ClassifyResult{ToolClaudeCode, CatConfig}
	case filename == "settings.json" && strings.Contains(p, "/.claude/"):
		return ClassifyResult{ToolClaudeCode, CatConfig}
	case filename == "settings.local.json" && strings.Contains(p, "/.claude/"):
		return ClassifyResult{ToolClaudeCode, CatConfig}
	case filename == ".claude.json":
		return ClassifyResult{ToolClaudeCode, CatConfig}
	case strings.Contains(p, "/.claude/agents/") && strings.HasSuffix(filename, ".md"):
		return ClassifyResult{ToolClaudeCode, CatAgent}
	case strings.Contains(p, "/.claude/commands/") && strings.HasSuffix(filename, ".md"):
		return ClassifyResult{ToolClaudeCode, CatCommand}
	case strings.Contains(p, "/.claude/plugins/") && strings.HasSuffix(filename, ".md"):
		return ClassifyResult{ToolClaudeCode, CatSkill}
	case strings.Contains(p, "/.claude/skills/") && filename == "skill.md":
		return ClassifyResult{ToolClaudeCode, CatSkill}

	// Gemini CLI
	case filename == "gemini.md":
		return ClassifyResult{ToolGeminiCLI, CatConfig}
	case filename == "settings.json" && strings.Contains(p, "/.gemini/"):
		return ClassifyResult{ToolGeminiCLI, CatConfig}
	case strings.Contains(p, "/.gemini/agents/") && strings.HasSuffix(filename, ".md"):
		return ClassifyResult{ToolGeminiCLI, CatAgent}

	// Kiro
	case strings.Contains(p, "/.kiro/steering/") && strings.HasSuffix(filename, ".md"):
		return ClassifyResult{ToolKiro, CatRule}
	case strings.Contains(p, "/.kiro/agents/") && strings.HasSuffix(filename, ".json"):
		return ClassifyResult{ToolKiro, CatAgent}
	case filename == "mcp.json" && strings.Contains(p, "/.kiro/settings/"):
		return ClassifyResult{ToolKiro, CatConfig}

	// Cursor
	case filename == ".cursorrules" || filename == ".cursorignore":
		return ClassifyResult{ToolCursor, CatRule}
	case strings.Contains(p, "/.cursor/rules/") && strings.HasSuffix(filename, ".mdc"):
		return ClassifyResult{ToolCursor, CatRule}
	case filename == "mcp.json" && strings.Contains(p, "/.cursor/"):
		return ClassifyResult{ToolCursor, CatConfig}

	// Windsurf
	case filename == ".windsurfrules":
		return ClassifyResult{ToolWindsurf, CatRule}

	// GitHub Copilot
	case filename == "copilot-instructions.md" && strings.Contains(p, "/.github/"):
		return ClassifyResult{ToolCopilot, CatConfig}
	case strings.Contains(p, "/.github/agents/") && strings.HasSuffix(filename, ".md"):
		return ClassifyResult{ToolCopilot, CatAgent}

	// Aider
	case filename == ".aider.conf.yml":
		return ClassifyResult{ToolAider, CatConfig}
	case filename == "aider.conventions.md":
		return ClassifyResult{ToolAider, CatRule}

	// Amp
	case strings.Contains(p, "/.amp/skills/") && strings.HasSuffix(filename, ".md"):
		return ClassifyResult{ToolAmp, CatSkill}
	case filename == "settings.json" && (strings.Contains(p, "/.amp/") || strings.Contains(p, "/.config/amp/")):
		return ClassifyResult{ToolAmp, CatConfig}

	// Codex/OpenAI
	case strings.Contains(p, "/.codex/agents/") && strings.HasSuffix(filename, ".toml"):
		return ClassifyResult{ToolCodex, CatAgent}
	case filename == "config.toml" && strings.Contains(p, "/.codex/"):
		return ClassifyResult{ToolCodex, CatConfig}
	case filename == "agents.md":
		return ClassifyResult{ToolCodex, CatConfig}
	}
	return ClassifyResult{ToolUnknown, CatConfig}
}
