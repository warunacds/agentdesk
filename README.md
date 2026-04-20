# Agent Desk

A free, open-source desktop app for managing AI agent skills, agents, commands, and MCP servers across Claude Code, Cursor, Gemini CLI, Kiro, Amp, Codex, and more — all in one place.

[agentdesk.sh](https://agentdesk.sh) · [Download](https://agentdesk.sh/download) · [User guide](https://agentdesk.sh/help) · [Discord](https://discord.gg/5s7mNS76) · [Report an issue](https://github.com/warunacds/agentdesk/issues)

## What it does

Agent Desk scans the standard locations for AI tool configuration (`~/.claude/skills/`, `~/.cursor/rules/`, `~/.kiro/steering/`, etc.) and gives you a single editor to:

- **Browse** every skill, agent, command, and rule, grouped by tool
- **Edit** markdown skills with syntax highlighting and preview
- **Create** new skills with the correct frontmatter and file location for each tool
- **Organize** skills into collections and project scopes
- **Enable/disable** skills without deleting them
- **Install** community skills from a built-in marketplace
- **Inspect** MCP servers and Claude Code plugins

Skill files stay in their original tool-specific locations — Agent Desk reads and edits them in place. Nothing is moved or copied.

## Install

**macOS (Homebrew):**

```bash
brew install --cask warunacds/tap/agentdesk
```

**Any platform:** download a pre-built binary from [agentdesk.sh/download](https://agentdesk.sh/download).

Supported platforms: macOS 11+ (arm64, Intel), Linux (x86_64), Windows (x86_64).

## Build from source

Requirements:

- Go 1.22+
- Node.js 24+
- [Wails v2 CLI](https://wails.io/docs/gettingstarted/installation)

```bash
# Dev build with hot reload
wails dev

# Production build (writes to build/bin/)
wails build
```

## Data

All app state lives in `~/.agentdesk/`:

```
~/.agentdesk/
  collections.json
  settings.json
  disabled-skills.json
  recent.json
  identity.key         # libp2p keypair for P2P sync (beta)
  peers.json
  sync-state.json
```

Your actual skill files stay where each tool expects them:

| Tool | Scanned directories |
|---|---|
| Claude Code | `~/.claude/agents/`, `~/.claude/commands/`, `~/.claude/skills/` |
| Gemini CLI | `~/.gemini/agents/` |
| Cursor | `~/.cursor/rules/` |
| Kiro | `~/.kiro/steering/`, `~/.kiro/agents/` |
| Amp | `~/.amp/skills/` |
| Codex | `~/.codex/agents/` |

You can add custom paths in **Settings → Scan Paths** for project-level or shared-folder setups.

## Architecture

- **Backend:** Go 1.22, Wails v2 runtime
- **Frontend:** React 18, TypeScript, Vite, CodeMirror 6
- **File watching:** fsnotify
- **P2P sync (beta):** libp2p, go-libp2p-kad-dht
- **Package layout:**
  - `app.go` — Wails-bound methods on the `App` struct
  - `internal/skills/` — scanning, classifying, reading, writing, collections, MCP, plugins, marketplace
  - `internal/sync/` — P2P sync engine and libp2p host
  - `internal/team/` — team sync (managed config distribution)
  - `cmd/agentdesk-core/` — standalone CLI binary that shares the same packages
  - `frontend/` — Wails-bundled React UI

## Contributing

Pull requests welcome. Small focused changes are easier to review than large refactors.

Before submitting:

```bash
go build ./...
go vet ./...
cd frontend && npm run build
```

For bugs and feature requests, use [the issue tracker](https://github.com/warunacds/agentdesk/issues). For quick questions, [Discord](https://discord.gg/5s7mNS76) is faster.

## License

MIT. See [LICENSE](./LICENSE).
