# Agent Desk — Claude Code Project Instructions

## What is Agent Desk

A cross-platform desktop app for managing AI agent configuration files (agents, skills, commands, rules, MCP servers, plugins) across multiple AI coding tools. Go backend + React + TypeScript frontend via Wails v2. Free and open source.

## Tech Stack

- **Backend:** Go 1.22+, Wails v2 (desktop framework)
- **Frontend:** React 18, TypeScript, Vite, CodeMirror 6
- **P2P Sync (beta):** libp2p v0.38.2, go-libp2p-kad-dht v0.28.2
- **File watching:** fsnotify
- **Module name:** `agentdesk` (import paths: `agentdesk/internal/skills`, `agentdesk/internal/sync`)

## Build Commands

```bash
# Development with hot reload
wails dev

# Production build
wails build

# Go only (check compilation)
go build ./...
go vet ./...

# Frontend type check
cd frontend && npm run build
```

## Project Structure

```
agentdesk/
├── main.go                       # Wails entry point, window config
├── app.go                        # App struct with ALL Wails-bound methods
├── wails.json                    # Wails project config
├── go.mod                        # Module: agentdesk
├── cmd/
│   └── agentdesk-core/           # Standalone CLI (shares internal packages)
├── internal/
│   ├── skills/                   # Core skill management
│   │   ├── types.go              # Skill, ToolType, Category, ToolMeta
│   │   ├── classifier.go         # File path → tool + category classification
│   │   ├── scanner.go            # Directory scanning, frontmatter parsing
│   │   ├── watcher.go            # fsnotify file watcher
│   │   ├── collections.go        # Collection CRUD
│   │   ├── settings.go           # App settings
│   │   ├── marketplace.go        # GitHub API marketplace integration
│   │   ├── mcp.go                # MCP server config parser
│   │   ├── plugins.go            # Plugin manifest parser (Claude Code)
│   │   ├── disabled.go           # Enable/disable skills via .disabled suffix
│   │   ├── projects.go           # Per-project scopes
│   │   └── recent.go             # Recently opened files
│   ├── sync/                     # P2P sync (beta)
│   │   ├── identity.go           # Ed25519 keypair, human-readable device ID
│   │   ├── node.go               # libp2p host, mDNS + DHT discovery
│   │   ├── peers.go              # Paired peer storage
│   │   ├── engine.go             # Sync orchestrator, message handling
│   │   ├── state.go              # Version vectors, SHA-256 tracking
│   │   ├── conflicts.go          # Version vector comparison
│   │   ├── manifest.go           # Selective sync manifest
│   │   └── sync.go               # Message types, protocol, NodeHost interface
│   └── team/                     # Team sync (managed config distribution)
└── frontend/
    └── src/
        ├── main.tsx              # React entry with ErrorBoundary
        ├── App.tsx               # Root component, all state management
        ├── types.ts              # All TypeScript interfaces
        ├── index.css             # Design tokens, dark/light themes
        └── components/
            ├── Sidebar.tsx       # Resizable sidebar, collections, tool groups
            ├── SkillEditor.tsx   # CodeMirror 6 editor + markdown toolbar + preview
            ├── MetadataPanel.tsx # Frontmatter form editor
            ├── NewSkillModal.tsx # Create skill dialog
            ├── SettingsPanel.tsx # Preferences, scan paths
            ├── SyncPanel.tsx     # P2P peer management UI
            ├── UpgradePrompt.tsx # P2P "Coming Soon" placeholder
            ├── MarketplacePanel.tsx
            ├── MCPPanel.tsx
            ├── PluginsPanel.tsx
            ├── ErrorBoundary.tsx
            └── Toast.tsx
```

## Key Conventions

### Go Backend

- **All Wails-bound methods live in `app.go`** on the `App` struct. Every exported method on `App` becomes callable from the frontend.
- **Package aliases:** import `syncpkg "agentdesk/internal/sync"` in `app.go` to avoid conflict with stdlib `sync`.
- **Error handling:** Functions that read config files return empty defaults on error — never crash.
- **Data directory:** all app data lives in `~/.agentdesk/`. A one-time migration from the old `~/.forge/` path runs on startup (see `migrateDataDir()` in `app.go`).
- **Device ID:** `GetDeviceID()` returns a human-readable identifier (`agentdesk-word-word-word`) derived from the libp2p peer ID.

### Frontend

- **All state lives in `App.tsx`.** Components receive data and callbacks via props. No context providers or state management libraries.
- **Panels are mutually exclusive:** only one of Settings, Marketplace, MCP, Plugins, or SkillEditor is visible at a time.
- **Wails bindings** are auto-generated at `frontend/wailsjs/go/main/App.{ts,js}` when you run `wails build` or `wails dev`. Never hand-edit.
- **CSS design tokens** are in `index.css` as CSS custom properties. Dark theme `:root`, light theme `[data-theme="light"]`.
- **Inline styles only.** No CSS modules, no styled-components. All styling via React `style` props using CSS variables.
- **Font:** IBM Plex Sans for UI, IBM Plex Mono for code/paths.

### Naming

- **Go types:** `ToolType`, `Category`, `Skill`, `Collection`, `Settings`, `MCPServer`, `PluginInfo`
- **Frontend types:** mirror Go types in `types.ts`
- **Categories:** `agent`, `skill`, `command`, `rule`, `config`
- **Tool IDs:** `claude-code`, `gemini-cli`, `cursor`, `kiro`, `windsurf`, `copilot`, `aider`, `amp`, `codex`

## Data Flow

```
User action → React component → Wails binding → Go method → Filesystem
                                                              ↓
Filesystem change → fsnotify → EventsEmit → EventsOn → Re-render
                             → Sync Engine → Broadcast to peers
```

## Supported Tools & File Locations

| Tool | Scanned directories |
|---|---|
| Claude Code | `~/.claude/agents/`, `~/.claude/commands/`, `~/.claude/skills/` |
| Gemini CLI | `~/.gemini/agents/` |
| Cursor | `~/.cursor/rules/` |
| Kiro | `~/.kiro/steering/`, `~/.kiro/agents/` |
| Windsurf | (project-level only) |
| Copilot | (project-level only) |
| Aider | (project-level only) |
| Amp | `~/.amp/skills/` |
| Codex | `~/.codex/agents/` |

Users can add project directories via **Settings → Scan Paths**. The scanner recurses up to 8 levels deep and skips noise folders (`node_modules`, `.git`, `vendor`, `dist`, `build`, `.next`, `__pycache__`).

## Things to Watch Out For

- **Never scan `~` directly** — causes macOS permission prompts for Documents/Desktop. Only scan specific known subdirectories.
- **libp2p is pinned to v0.38.2** for Go 1.22 compat. Don't upgrade without testing.
- **`internal/sync` package name shadows stdlib `sync`** — always use `gosync "sync"` or `syncpkg "agentdesk/internal/sync"` aliases where needed.
- **Env values in MCP configs are masked server-side** in Go before reaching the frontend. Never expose API keys.
- **README.md files are filtered out** in the scanner (`readme.md` check in `collectFile`).
- **File names in sidebar strip extensions** — `d.Name()` without extension is used as the display name when frontmatter has no `name` field.

## Release

Tags on `main` trigger GitHub Actions to build all platforms, sign + notarize macOS DMGs, upload to Cloudflare R2, update the Homebrew cask, and rewrite the download page on the website.

```bash
git tag vX.Y.Z-beta
git push origin vX.Y.Z-beta
```
