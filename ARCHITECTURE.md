# Forge Architecture

## System Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Forge Desktop App                            │
│                                                                     │
│  ┌─────────────────────┐    Wails IPC    ┌────────────────────────┐ │
│  │    Go Backend        │◄──────────────►│   React Frontend        │ │
│  │                      │   (JSON RPC)   │                        │ │
│  │  ┌────────────────┐  │                │  ┌──────────────────┐  │ │
│  │  │ internal/skills │  │                │  │    App.tsx        │  │ │
│  │  │                 │  │                │  │  (state manager)  │  │ │
│  │  │ Scanner         │  │                │  └────────┬─────────┘  │ │
│  │  │ Classifier      │  │                │           │            │ │
│  │  │ Watcher         │  │    Events      │  ┌────────┴─────────┐  │ │
│  │  │ Collections     │◄─────────────────►│  │   Components     │  │ │
│  │  │ Settings        │  │ (file-changed, │  │                  │  │ │
│  │  │ Marketplace     │  │  sync-*, etc)  │  │  Sidebar         │  │ │
│  │  │ MCP             │  │                │  │  SkillEditor     │  │ │
│  │  │ Plugins         │  │                │  │  SettingsPanel   │  │ │
│  │  └────────────────┘  │                │  │  MarketplacePanel│  │ │
│  │                      │                │  │  MCPPanel        │  │ │
│  │  ┌────────────────┐  │                │  │  PluginsPanel    │  │ │
│  │  │ internal/sync   │  │                │  │  SyncPanel       │  │ │
│  │  │                 │  │                │  │  MetadataPanel   │  │ │
│  │  │ Node (libp2p)   │  │                │  │  Toast           │  │ │
│  │  │ Engine          │  │                │  │  ErrorBoundary   │  │ │
│  │  │ Peers           │  │                │  └──────────────────┘  │ │
│  │  │ State           │  │                │                        │ │
│  │  └────────────────┘  │                │  ┌──────────────────┐  │ │
│  │                      │                │  │  CodeMirror 6    │  │ │
│  └─────────────────────┘                │  │  (editor engine) │  │ │
│                                          │  └──────────────────┘  │ │
│                                          └────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
         │                                              │
         │  Filesystem                                  │  Webview
         ▼                                              ▼
┌─────────────────┐  ┌──────────────┐  ┌──────────────────────────────┐
│ ~/.claude/       │  │ ~/.forge/     │  │ macOS: WKWebView            │
│ ~/.gemini/       │  │ collections  │  │ Linux: WebKitGTK             │
│ ~/.cursor/       │  │ settings     │  │ Windows: WebView2            │
│ ~/.kiro/         │  │ identity.key │  └──────────────────────────────┘
│ ~/.amp/          │  │ peers.json   │
│ ~/.codex/        │  │ sync-state   │
└─────────────────┘  └──────────────┘
```

## Go Backend Architecture

### Package: `internal/skills`

Handles all skill file operations. No network, no UI — pure filesystem and data logic.

```
internal/skills/
├── types.go          Core types: Skill, ToolType, Category, ToolMeta, KnownTools
├── classifier.go     Classify() — file path → {Tool, Category}
├── scanner.go        Scan() — walk directories, collect skills, parse frontmatter
├── watcher.go        Watcher — fsnotify wrapper with callback
├── collections.go    CRUD for ~/.forge/collections.json
├── settings.go       CRUD for ~/.forge/settings.json
├── marketplace.go    GitHub Trees API, caching, skill installation
├── mcp.go            Parse MCP configs from 8 tools
└── plugins.go        Parse installed_plugins.json from Claude Code
```

**Scanner flow:**
```
Scan(extraDirs)
  → Build knownRoots list (tool-specific subdirs only)
  → Pick up top-level config files (CLAUDE.md, GEMINI.md)
  → For each root: scanDeep() → collectFile() → Classify()
  → Deduplicate by real path (symlink resolution)
  → Sort by modification time (newest first)
  → Return []Skill
```

**Classifier logic:**
```
Classify(absPath) → {Tool, Category}

Path contains /.claude/agents/  → {claude-code, agent}
Path contains /.claude/commands/ → {claude-code, command}
Filename is .cursorrules        → {cursor, rule}
Path contains /.kiro/agents/    → {kiro, agent}
...etc (see classifier.go for full rules)
```

### Package: `internal/sync`

P2P file synchronization via libp2p. Currently in beta.

```
internal/sync/
├── sync.go          Message types, NodeHost interface, protocol helpers
├── identity.go      Ed25519 keypair generation and persistence
├── node.go          libp2p host, mDNS + DHT discovery, SyncEngine interface
├── peers.go         PeerStore — paired peer persistence
├── state.go         SyncState — version vectors, file hashes
├── conflicts.go     CompareVersions(), MergeVersions()
├── engine.go        Engine — orchestrates sync, handles all message types
└── watched.go       SyncDirs(), IsSyncable(), ScanSyncableFiles()
```

**Key interfaces:**

```go
// NodeHost — what the Engine needs from the Node (defined in sync.go)
type NodeHost interface {
    PeerID() peer.ID
    HumanID() string
    Connectedness(pid peer.ID) network.Connectedness
    SendToPeer(pid peer.ID, data []byte) error
    BroadcastToPairedPeers(data []byte)
    EmitEvent(event string, data map[string]string)
}

// SyncEngine — what the Node needs from the Engine (defined in node.go)
type SyncEngine interface {
    IsPairedPeer(pid peer.ID) bool
    GetPairedPeers() []PeerInfo
    HandleIncoming(from peer.ID, data []byte, s network.Stream)
}
```

**Sync protocol flow:**
```
File changes locally
  → fsnotify fires
  → Engine.OnLocalFileChange()
  → Compute SHA-256 hash
  → Increment version vector for local peer
  → Save sync state
  → Broadcast FileAnnounce to paired peers

Peer receives FileAnnounce
  → CompareVersions(local, remote)
  → LocalNewer: ignore
  → RemoteNewer: request FileData, write to disk
  → Concurrent: archive local as .conflict-*, request remote, notify UI
  → Equal: ignore
```

**Discovery mechanisms:**
```
1. mDNS (LAN)     → instant, zero config
2. DHT (WAN)      → Kademlia via IPFS bootstrap nodes
3. Relay (fallback) → IPFS relay nodes, E2E encrypted
```

### Entry point: `app.go`

The `App` struct is the bridge between Go and the frontend. All exported methods become Wails bindings.

```go
type App struct {
    ctx        context.Context
    mu         sync.Mutex
    extraDirs  []string
    watcher    *skills.Watcher
    syncNode   *forgesync.Node
    syncEngine *forgesync.Engine
}
```

**Lifecycle:**
```
startup(ctx)
  → Create file watcher, start watching
  → Create libp2p Node
  → Create sync Engine, wire to Node
  → Register sync protocol handler
  → Start mDNS + DHT discovery
  → Start Engine (reconcile local files)

shutdown(ctx)
  → Stop Engine (save state)
  → Stop Node (close libp2p host)
  → Stop Watcher
```

**Bound methods (callable from frontend):**

| Category | Methods |
|---|---|
| Skills | `GetSkills`, `SaveSkill`, `CreateSkill`, `DeleteSkill`, `UpdateSkillMetadata` |
| Collections | `GetCollections`, `CreateCollection`, `RenameCollection`, `DeleteCollection`, `AddToCollection`, `RemoveFromCollection` |
| Settings | `GetSettings`, `SaveSettings`, `GetDefaultScanPaths`, `SelectDirectoryForTool` |
| Marketplace | `GetMarketplaceSources`, `BrowseMarketplace`, `PreviewMarketplaceSkill`, `InstallMarketplaceSkill` |
| MCP | `GetMCPConfigs` |
| Plugins | `GetPlugins` |
| Sync | `GetSyncStatus`, `GetForgeID`, `GetFullPeerID`, `GetPeers`, `AddPeerByID`, `RemovePeerByID` |
| Utility | `GetKnownTools`, `GetDefaultFilename`, `GetDefaultSubdir`, `RevealInFinder`, `SelectDirectory`, `AddDirectory`, `RemoveDirectory` |

## Frontend Architecture

### State Management

All state lives in `App.tsx`. No Redux, no Zustand, no Context — just `useState` + `useCallback`.

```
App.tsx state:
├── skills: Skill[]              — all scanned skills
├── tools: ToolMeta[]            — known tool metadata
├── collections: Collection[]    — user collections
├── settings: Settings           — user preferences
├── syncStatus: SyncStatus       — P2P sync state
├── mcpConfigs: MCPToolConfig[]  — MCP server configs
├── pluginConfigs: PluginToolConfig[] — installed plugins
├── activeId: string | null      — selected skill
├── activeCollection: string     — selected collection filter
├── showSettings/Marketplace/MCP/Plugins: boolean — panel visibility
├── toast: ToastState | null     — undo notification
└── loading: boolean             — initial scan in progress
```

### Component Tree

```
ErrorBoundary
└── App
    ├── Titlebar (inline in App)
    ├── Sidebar
    │   ├── Search input
    │   ├── Collections section
    │   │   └── Collection items (with context menus)
    │   ├── Tool groups
    │   │   └── Category subgroups
    │   │       └── Skill items (with context menus)
    │   └── Footer (+ New Skill, Marketplace, MCP, Plugins, Settings)
    ├── Main content area (one of):
    │   ├── PluginsPanel
    │   ├── MCPPanel
    │   ├── MarketplacePanel
    │   ├── SettingsPanel
    │   │   └── SyncPanel
    │   ├── SkillEditor
    │   │   ├── Header (tool badges, name, save button)
    │   │   ├── MetadataPanel (collapsible frontmatter form)
    │   │   ├── MarkdownToolbar
    │   │   ├── CodeMirror 6 editor
    │   │   └── Status bar
    │   └── Onboarding (welcome screen)
    ├── NewSkillModal (overlay)
    └── Toast (fixed position, bottom center)
```

### Panel Mutual Exclusion

Only one panel is visible at a time. Opening one closes all others:

```typescript
// Pattern used everywhere:
setShowSettings(true);
setShowMarketplace(false);
setShowMCP(false);
setShowPlugins(false);
```

### Event System

Go → Frontend communication via Wails events:

| Event | Trigger | Handler |
|---|---|---|
| `file-changed` | fsnotify detects a file change | Reloads skill list |
| `sync-discovery` | mDNS/DHT discovers a peer | Updates sync status |
| `file-synced` | Remote file written to disk | Reloads skill list |
| `sync-conflict` | Version vector conflict detected | Shows notification |

### Theming

Two themes defined in `index.css` via CSS custom properties:

```css
:root { /* Dark theme (default) */ }
[data-theme="light"] { /* Light overrides */ }
```

Theme applied by: `document.documentElement.setAttribute('data-theme', theme)`

CodeMirror has separate theme objects (`forgeDarkTheme`, `forgeLightTheme`) that map CSS variables to CodeMirror styling.

## Data Persistence

All data stored in `~/.forge/`:

```
~/.forge/
├── collections.json     — user-created skill groupings
├── settings.json        — preferences, custom scan paths, theme
├── identity.key         — Ed25519 private key for P2P (0600 perms)
├── peers.json           — paired device list
├── sync-state.json      — version vectors and file hashes
└── trash/               — files deleted via sync (recoverable)
```

**Never synced:** `identity.key`, `peers.json`, `sync-state.json` (machine-specific)

## Network Architecture (P2P Sync)

```
Machine A                              Machine B
┌──────────────┐                     ┌──────────────┐
│ Forge App    │                     │ Forge App    │
│              │                     │              │
│ Engine       │◄── libp2p stream ──►│ Engine       │
│  ↑           │   /forge/sync/1.0.0 │           ↑  │
│  │ fsnotify  │                     │  fsnotify │  │
│  ↓           │                     │           ↓  │
│ ~/.claude/   │                     │ ~/.claude/   │
│ ~/.cursor/   │                     │ ~/.cursor/   │
└──────────────┘                     └──────────────┘
        │                                    │
        ▼                                    ▼
   ┌─────────┐                         ┌─────────┐
   │  mDNS   │ (LAN discovery)        │  mDNS   │
   │  DHT    │ (WAN discovery)        │  DHT    │
   │  Relay  │ (NAT fallback)         │  Relay  │
   └─────────┘                         └─────────┘
```

**Message types on the wire:**
```
FileAnnounce  → "I changed this file" (hash + version vector)
FileRequest   → "Send me that file"
FileData      → "Here's the file content" (base64)
FileDelete    → "I deleted this file" (tombstone)
SyncRequest   → "Send me your full state"
SyncState     → "Here's everything I have"
```

**Conflict resolution:**
```
CompareVersions(local, remote) →
  LocalNewer:  ignore (we're ahead)
  RemoteNewer: accept remote version
  Concurrent:  archive local as .conflict-*, accept remote, notify user
  Equal:       no action needed
```

## Build & Release Pipeline

```
git tag -a v0.2.0 -m "description"
git push origin v0.2.0
    │
    ▼
GitHub Actions (.github/workflows/release.yml)
    │
    ├── build-linux (ubuntu-latest)
    │   └── wails build -tags webkit2_41 → Forge-linux-amd64.tar.gz
    │
    ├── build-macos (macos-latest)
    │   ├── wails build darwin/arm64 → Forge-macOS-arm64.dmg
    │   └── wails build darwin/amd64 → Forge-macOS-intel.dmg
    │
    └── build-windows (windows-latest)
        └── wails build windows/amd64 → Forge-windows-amd64.zip
    │
    ▼
GitHub Release (auto-created with all 4 binaries)
    │
    ▼
update-homebrew.yml → updates warunacds/homebrew-tap formula
```

**Tags with `beta`, `alpha`, or `rc` are marked as prerelease.**
