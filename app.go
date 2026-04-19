package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"agentdesk/internal/skills"
	syncpkg "agentdesk/internal/sync"
	"agentdesk/internal/team"
)

type App struct {
	ctx        context.Context
	mu         sync.Mutex
	extraDirs  []string
	watcher    *skills.Watcher
	syncNode   *syncpkg.Node
	syncEngine *syncpkg.Engine
	teamSync   *team.TeamSync
	teamClient *team.Client
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	migrateDataDir()
	wt, err := skills.NewWatcher(func(path, op string) {
		runtime.EventsEmit(ctx, "file-changed", map[string]string{"path": path, "op": op})
		if a.syncEngine != nil {
			a.syncEngine.OnLocalFileChange(path)
		}
	})
	if err == nil {
		wt.Start()
		a.watcher = wt
	}

	// P2P sync auto-starts for everyone (free, open source).
	a.startSync()

	// Start team sync if previously connected
	a.startTeamSyncIfConnected()
}

// startSync initialises the P2P sync node and engine. It is safe to call
// multiple times — it no-ops if the node is already running.
func (a *App) startSync() {
	if a.syncNode != nil {
		return
	}
	node, err := syncpkg.NewNode()
	if err != nil {
		log.Printf("sync: failed to create node: %v", err)
		return
	}
	a.syncNode = node
	engine := syncpkg.NewEngine(node)
	a.syncEngine = engine
	node.SetEngine(engine)
	node.RegisterProtocol()
	node.SetOnDiscovery(func(evt syncpkg.DiscoveryEvent) {
		runtime.EventsEmit(a.ctx, "sync-discovery", map[string]string{
			"peerId": evt.PeerID,
			"event":  evt.Event,
		})
	})
	if err := node.StartDiscovery(); err != nil {
		log.Printf("sync: discovery start error: %v", err)
	}
	engine.Start()
}

func (a *App) shutdown(_ context.Context) {
	if a.teamSync != nil {
		a.teamSync.Stop()
	}
	if a.syncEngine != nil {
		a.syncEngine.Stop()
	}
	if a.syncNode != nil {
		a.syncNode.Stop()
	}
	if a.watcher != nil {
		a.watcher.Stop()
	}
}

func (a *App) GetSkills() []skills.Skill {
	a.mu.Lock()
	extra := append([]string{}, a.extraDirs...)
	a.mu.Unlock()

	// Merge custom paths from settings into extra dirs
	settings := skills.LoadSettings()
	for _, paths := range settings.CustomPaths {
		extra = append(extra, paths...)
	}

	return skills.Scan(extra)
}

func (a *App) SaveSkill(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func (a *App) CreateSkill(path string, tool skills.ToolType, name string) (skills.Skill, error) {
	tmpl := skills.GenerateTemplate(tool, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return skills.Skill{}, err
	}
	if err := os.WriteFile(path, []byte(tmpl), 0644); err != nil {
		return skills.Skill{}, err
	}
	// Resolve the real path for the newly created file
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		realPath = path
	}

	return skills.Skill{
		ID: uuid.New().String(), Name: name, Path: path, RealPath: realPath,
		Tool: tool, Tools: []skills.ToolType{tool},
		Content: tmpl, Frontmatter: map[string]string{"name": name},
		Modified: skills.NowUnix(), Size: int64(len(tmpl)),
	}, nil
}

func (a *App) DeleteSkill(path string) error { return os.Remove(path) }

// ReadSkillFileResult is the response for ReadSkillFile.
type ReadSkillFileResult struct {
	Content  string `json:"content"`
	Size     int64  `json:"size"`
	Language string `json:"language"`
}

// ReadSkillFile reads a file under the user's home directory.
func (a *App) ReadSkillFile(path string) (*ReadSkillFileResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	absHome, _ := filepath.Abs(home)
	if !strings.HasPrefix(absPath, absHome+string(os.PathSeparator)) && absPath != absHome {
		return nil, fmt.Errorf("path is outside user home")
	}
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	return &ReadSkillFileResult{
		Content:  string(content),
		Size:     int64(len(content)),
		Language: classifyLanguageForReadFile(absPath),
	}, nil
}

func classifyLanguageForReadFile(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".markdown", ".mdc", ".mdx":
		return "markdown"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".sh", ".bash", ".zsh":
		return "shell"
	case ".go":
		return "go"
	case ".rb":
		return "ruby"
	case ".rs":
		return "rust"
	default:
		return "text"
	}
}

// emitSkillsChanged notifies the frontend that the set of enabled/disabled
// skills has changed so it can refresh its view.
func (a *App) emitSkillsChanged() {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "skills-changed")
	}
}

// DisableSkill moves a skill file to ~/.agentdesk/disabled/<tool>/ so tools stop seeing it.
func (a *App) DisableSkill(path string, tool string) error {
	if err := skills.DisableSkill(path, skills.ToolType(tool)); err != nil {
		return err
	}
	a.emitSkillsChanged()
	return nil
}

// EnableSkill restores a previously disabled skill to its original path.
func (a *App) EnableSkill(originalPath string) error {
	if err := skills.EnableSkill(originalPath); err != nil {
		return err
	}
	a.emitSkillsChanged()
	return nil
}

// IsSkillDisabled returns true if a skill path is currently in the disabled store.
func (a *App) IsSkillDisabled(originalPath string) bool {
	return skills.IsDisabled(originalPath)
}

// ListDisabledSkills returns all currently disabled skills with metadata.
func (a *App) ListDisabledSkills() []skills.DisabledSkillInfo {
	return skills.ListDisabledSkills()
}

// DisableSkillsForTool disables every skill currently scanned for a tool.
// Returns the number of skills disabled.
func (a *App) DisableSkillsForTool(tool string) (int, error) {
	current := a.GetSkills()
	tt := skills.ToolType(tool)
	count := 0
	for _, s := range current {
		if s.Tool != tt {
			continue
		}
		if skills.IsDisabled(s.Path) {
			continue
		}
		if err := skills.DisableSkill(s.Path, tt); err != nil {
			return count, err
		}
		count++
	}
	if count > 0 {
		a.emitSkillsChanged()
	}
	return count, nil
}

// EnableAllDisabledForTool restores every disabled skill belonging to a tool.
// Returns the number of skills enabled.
func (a *App) EnableAllDisabledForTool(tool string) (int, error) {
	tt := skills.ToolType(tool)
	list := skills.ListDisabledSkills()
	count := 0
	for _, e := range list {
		if e.Tool != tt {
			continue
		}
		if err := skills.EnableSkill(e.OriginalPath); err != nil {
			return count, err
		}
		count++
	}
	if count > 0 {
		a.emitSkillsChanged()
	}
	return count, nil
}

// ListProjects returns all pinned projects.
func (a *App) ListProjects() []skills.Project {
	return skills.LoadProjectStore().Projects
}

// AddProject pins a new project folder.
func (a *App) AddProject(path string, name string) (*skills.Project, error) {
	p, err := skills.AddProject(path, name)
	if err != nil {
		return nil, err
	}
	a.emitSkillsChanged()
	return &p, nil
}

// RemoveProject removes a pinned project by ID.
func (a *App) RemoveProject(id string) error {
	if err := skills.RemoveProject(id); err != nil {
		return err
	}
	a.emitSkillsChanged()
	return nil
}

// RenameProject changes a project's display name.
func (a *App) RenameProject(id string, name string) error {
	if err := skills.RenameProject(id, name); err != nil {
		return err
	}
	a.emitSkillsChanged()
	return nil
}

// ScanProjectSkills returns skills within a project's path.
func (a *App) ScanProjectSkills(path string) []skills.Skill {
	return skills.ScanProjectSkills(path)
}

// SetProjectSkillOverride disables or enables a global skill for a specific project.
func (a *App) SetProjectSkillOverride(projectPath string, skillPath string, disabled bool) error {
	if err := skills.SetProjectOverride(projectPath, skillPath, disabled); err != nil {
		return err
	}
	a.emitSkillsChanged()
	return nil
}

// GetProjectOverrides returns the list of disabled skill paths for a project.
func (a *App) GetProjectOverrides(projectPath string) []string {
	return skills.GetProjectOverrides(projectPath)
}

// SelectProjectDirectory shows a native folder picker.
func (a *App) SelectProjectDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Pick a project folder",
	})
}

// UpdateSkillMetadata reads the file at path, replaces the frontmatter section
// with the provided metadata map (preserving the body), and writes back.
// If the file has no frontmatter, it prepends one.
func (a *App) UpdateSkillMetadata(path string, metadata map[string]string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	updated := skills.ReplaceFrontmatter(string(data), metadata)
	return os.WriteFile(path, []byte(updated), 0644)
}

func (a *App) AddDirectory(path string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, d := range a.extraDirs {
		if d == path {
			return
		}
	}
	a.extraDirs = append(a.extraDirs, path)
	if a.watcher != nil {
		_ = a.watcher.Add(path)
	}
}

func (a *App) RemoveDirectory(path string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	filtered := a.extraDirs[:0]
	for _, d := range a.extraDirs {
		if d != path {
			filtered = append(filtered, d)
		}
	}
	a.extraDirs = filtered
}

func (a *App) GetExtraDirs() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string{}, a.extraDirs...)
}

func (a *App) GetKnownTools() []skills.ToolMeta { return skills.KnownTools }
func (a *App) GetDefaultFilename(tool skills.ToolType) string {
	return skills.DefaultFilename(tool)
}
func (a *App) GetDefaultSubdir(tool skills.ToolType) string {
	return skills.DefaultSubdir(tool)
}

func (a *App) RevealInFinder(path string) error {
	dir := filepath.Dir(path)
	var cmd string
	var args []string
	switch goruntime.GOOS {
	case "darwin":
		cmd, args = "open", []string{"-R", path}
	case "windows":
		cmd, args = "explorer", []string{"/select,", path}
	default:
		cmd, args = "xdg-open", []string{dir}
	}
	return exec.Command(cmd, args...).Start()
}

func (a *App) SelectDirectory() string {
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select directory",
	})
	if err != nil {
		return ""
	}
	return path
}

// --- Collections ---

func (a *App) GetCollections() []skills.Collection {
	store := skills.LoadCollections()
	return store.Collections
}

func (a *App) CreateCollection(name string) skills.Collection {
	a.mu.Lock()
	defer a.mu.Unlock()

	store := skills.LoadCollections()
	c := skills.NewCollection(name)
	store.Collections = append(store.Collections, c)
	_ = skills.SaveCollections(store)
	return c
}

func (a *App) RenameCollection(id, name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	store := skills.LoadCollections()
	idx := skills.FindCollectionIndex(&store, id)
	if idx < 0 {
		return fmt.Errorf("collection not found")
	}
	store.Collections[idx].Name = name
	store.Collections[idx].Modified = skills.NowUnix()
	return skills.SaveCollections(store)
}

func (a *App) DeleteCollection(id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	store := skills.LoadCollections()
	idx := skills.FindCollectionIndex(&store, id)
	if idx < 0 {
		return fmt.Errorf("collection not found")
	}
	store.Collections = append(store.Collections[:idx], store.Collections[idx+1:]...)
	return skills.SaveCollections(store)
}

func (a *App) AddToCollection(collectionId, skillPath string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	store := skills.LoadCollections()
	idx := skills.FindCollectionIndex(&store, collectionId)
	if idx < 0 {
		return fmt.Errorf("collection not found")
	}
	store.Collections[idx].AddPath(skillPath)
	return skills.SaveCollections(store)
}

func (a *App) RemoveFromCollection(collectionId, skillPath string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	store := skills.LoadCollections()
	idx := skills.FindCollectionIndex(&store, collectionId)
	if idx < 0 {
		return fmt.Errorf("collection not found")
	}
	store.Collections[idx].RemovePath(skillPath)
	return skills.SaveCollections(store)
}

// --- Marketplace ---

func (a *App) GetMarketplaceSources() []skills.MarketplaceSource {
	return skills.DefaultSources
}

func (a *App) BrowseMarketplace(sourceID string) ([]skills.MarketplaceSkill, error) {
	for _, src := range skills.DefaultSources {
		if src.ID == sourceID {
			return skills.FetchMarketplaceSkills(src)
		}
	}
	return nil, fmt.Errorf("unknown source: %s", sourceID)
}

func (a *App) PreviewMarketplaceSkill(downloadURL string) (string, error) {
	return skills.FetchSkillContent(downloadURL)
}

func (a *App) InstallMarketplaceSkill(downloadURL string, tool string, filename string) (string, error) {
	content, err := skills.FetchSkillContent(downloadURL)
	if err != nil {
		return "", fmt.Errorf("fetch skill: %w", err)
	}
	return skills.InstallSkill(content, skills.ToolType(tool), filename)
}

// --- MCP Servers ---

func (a *App) GetMCPConfigs() []skills.MCPToolConfig {
	return skills.LoadAllMCPConfigs()
}

// --- Plugins ---

func (a *App) GetPlugins() []skills.PluginToolConfig {
	return skills.LoadAllPlugins()
}

// --- Settings ---

func (a *App) GetSettings() skills.Settings {
	return skills.LoadSettings()
}

func (a *App) SaveSettings(s skills.Settings) error {
	return skills.SaveSettings(s)
}

func (a *App) GetDefaultScanPaths() map[string][]string {
	return skills.DefaultScanPaths()
}

func (a *App) SelectDirectoryForTool() string {
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select directory to scan",
	})
	if err != nil {
		return ""
	}
	return path
}

// --- Recently Opened ---

// GetRecent returns the recently opened files store.
func (a *App) GetRecent() skills.RecentStore {
	return skills.LoadRecent()
}

// AddRecent records a file as recently opened.
func (a *App) AddRecent(path, name, tool string) {
	skills.AddRecent(path, name, tool)
}

// ClearRecent removes all recently opened file entries.
func (a *App) ClearRecent() error {
	return skills.ClearRecent()
}

// --- Duplicate ---

// DuplicateSkill reads the file at path, writes a copy with a "-copy" suffix,
// and returns the new skill. If "-copy" already exists, it appends "-copy-2",
// "-copy-3", etc.
func (a *App) DuplicateSkill(path string) (skills.Skill, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return skills.Skill{}, fmt.Errorf("read source: %w", err)
	}

	newPath := deriveCopyPath(path)

	dir := filepath.Dir(newPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return skills.Skill{}, fmt.Errorf("create directory: %w", err)
	}

	if err := os.WriteFile(newPath, content, 0644); err != nil {
		return skills.Skill{}, fmt.Errorf("write copy: %w", err)
	}

	realPath, err := filepath.EvalSymlinks(newPath)
	if err != nil {
		realPath = newPath
	}

	result := skills.Classify(newPath)
	fm, _ := skills.ParseFrontmatter(string(content))
	name := fm["name"]
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(newPath), filepath.Ext(newPath))
	}

	fi, err := os.Stat(newPath)
	if err != nil {
		return skills.Skill{}, fmt.Errorf("stat copy: %w", err)
	}

	return skills.Skill{
		ID:          uuid.New().String(),
		Name:        name,
		Path:        newPath,
		RealPath:    realPath,
		Tool:        result.Tool,
		Tools:       []skills.ToolType{result.Tool},
		Category:    result.Category,
		Content:     string(content),
		Frontmatter: fm,
		Modified:    fi.ModTime().Unix(),
		Size:        fi.Size(),
	}, nil
}

// deriveCopyPath generates a copy path: "foo.md" -> "foo-copy.md",
// and if that exists, "foo-copy-2.md", "foo-copy-3.md", etc.
func deriveCopyPath(original string) string {
	ext := filepath.Ext(original)
	base := strings.TrimSuffix(original, ext)

	candidate := base + "-copy" + ext
	if _, err := os.Stat(candidate); err != nil {
		return candidate
	}

	for i := 2; i < 100; i++ {
		candidate = fmt.Sprintf("%s-copy-%d%s", base, i, ext)
		if _, err := os.Stat(candidate); err != nil {
			return candidate
		}
	}

	return candidate
}

// --- Search ---

// SearchSkills performs a full-text search across all scanned skill files.
func (a *App) SearchSkills(query string) []skills.SearchResult {
	a.mu.Lock()
	extra := append([]string{}, a.extraDirs...)
	a.mu.Unlock()

	settings := skills.LoadSettings()
	for _, paths := range settings.CustomPaths {
		extra = append(extra, paths...)
	}

	return skills.SearchFiles(query, extra)
}

// --- Validation ---

// ValidateSkills runs validation checks on all scanned skills and returns
// warnings grouped by file path.
func (a *App) ValidateSkills() map[string][]skills.ValidationWarning {
	allSkills := a.GetSkills()
	return skills.ValidateAll(allSkills)
}

// --- Templates ---

// GetTemplates returns the list of built-in skill templates.
func (a *App) GetTemplates() []skills.SkillTemplate {
	return skills.BuiltinTemplates
}

// CreateFromTemplate creates a new skill file from a built-in template.
// The templateID selects the template, name overrides the frontmatter name,
// and path is the full destination file path.
func (a *App) CreateFromTemplate(templateID, name, path string) (skills.Skill, error) {
	tmpl := skills.FindTemplate(templateID)
	if tmpl == nil {
		return skills.Skill{}, fmt.Errorf("template not found: %s", templateID)
	}

	content := tmpl.Content

	// Replace the name in frontmatter if the content has one
	if name != "" && strings.Contains(content, "---") {
		fm, body := skills.ParseFrontmatter(content)
		fm["name"] = name
		content = skills.ReplaceFrontmatter(body, fm)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return skills.Skill{}, fmt.Errorf("create directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return skills.Skill{}, fmt.Errorf("write file: %w", err)
	}

	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		realPath = path
	}

	fm, _ := skills.ParseFrontmatter(content)
	displayName := fm["name"]
	if displayName == "" {
		displayName = name
	}

	return skills.Skill{
		ID:          uuid.New().String(),
		Name:        displayName,
		Path:        path,
		RealPath:    realPath,
		Tool:        tmpl.Tool,
		Tools:       []skills.ToolType{tmpl.Tool},
		Content:     content,
		Frontmatter: fm,
		Modified:    skills.NowUnix(),
		Size:        int64(len(content)),
	}, nil
}

// --- P2P Sync ---

// GetSyncStatus returns a summary of the sync engine's current state. If sync
// failed to initialise the returned map contains only {"running": false}.
func (a *App) GetSyncStatus() map[string]interface{} {
	if a.syncEngine == nil {
		return map[string]interface{}{"running": false}
	}
	return a.syncEngine.GetSyncStatus()
}

// GetDeviceID returns the human-readable device identifier for this device. It
// returns an empty string when sync is not running.
func (a *App) GetDeviceID() string {
	if a.syncNode == nil {
		return ""
	}
	return a.syncNode.HumanID()
}

// GetPeers returns information about every paired peer. It returns nil when
// sync is not running.
func (a *App) GetPeers() []syncpkg.PeerInfo {
	if a.syncEngine == nil {
		return nil
	}
	return a.syncEngine.GetPairedPeers()
}

// AddPeerByID pairs with the peer identified by peerIDStr. The name is a
// human-friendly label chosen by the user.
func (a *App) AddPeerByID(peerIDStr string, name string) error {
	if a.syncEngine == nil {
		return fmt.Errorf("sync not running")
	}
	pid, err := peer.Decode(peerIDStr)
	if err != nil {
		return fmt.Errorf("invalid peer ID: %w", err)
	}
	if err := a.syncEngine.AddPeer(pid, name); err != nil {
		return err
	}
	// Actively try to connect to the new peer
	go a.syncNode.ConnectToPeer(pid)
	return nil
}

// RemovePeerByID unpairs the peer with the given string ID.
func (a *App) RemovePeerByID(peerID string) error {
	if a.syncEngine == nil {
		return fmt.Errorf("sync not running")
	}
	return a.syncEngine.RemovePeer(peerID)
}

// GetFullPeerID returns the full libp2p peer ID string for this device. It
// returns an empty string when sync is not running.
func (a *App) GetFullPeerID() string {
	if a.syncNode == nil {
		return ""
	}
	return a.syncNode.PeerID().String()
}

// --- Selective Sync ---

// GetSyncManifest returns the current sync manifest. If sync is not running,
// a default empty manifest is returned.
func (a *App) GetSyncManifest() interface{} {
	if a.syncEngine == nil {
		return syncpkg.LoadManifest()
	}
	return a.syncEngine.GetManifest()
}

// AddToSyncManifest adds the given paths to the sync manifest so they will
// be included in future syncs. Returns an error if sync is not running.
func (a *App) AddToSyncManifest(paths []string) error {
	if a.syncEngine == nil {
		return fmt.Errorf("sync not running")
	}
	return a.syncEngine.AddToManifest(paths)
}

// RemoveFromSyncManifest removes the given paths from the sync manifest so
// they will no longer be synced. Returns an error if sync is not running.
func (a *App) RemoveFromSyncManifest(paths []string) error {
	if a.syncEngine == nil {
		return fmt.Errorf("sync not running")
	}
	return a.syncEngine.RemoveFromManifest(paths)
}

// SyncFiles manually triggers sync for the given file paths. Each path
// should be a tilde-relative path (e.g. "~/.claude/agents/foo.md").
// Returns an error if sync is not running.
func (a *App) SyncFiles(paths []string) error {
	if a.syncEngine == nil {
		return fmt.Errorf("sync not running")
	}
	return a.syncEngine.SyncFiles(paths)
}

// GetSyncConflicts returns all unresolved sync conflicts. Returns nil when
// sync is not running.
func (a *App) GetSyncConflicts() interface{} {
	if a.syncEngine == nil {
		return []syncpkg.SyncConflict{}
	}
	return a.syncEngine.GetConflicts()
}

// ResolveSyncConflict resolves a sync conflict for the given path using the
// specified resolution strategy: "keep-local", "keep-remote", or "keep-both".
func (a *App) ResolveSyncConflict(path string, resolution string) error {
	if a.syncEngine == nil {
		return fmt.Errorf("sync not running")
	}
	return a.syncEngine.ResolveConflict(path, resolution)
}

// --- Team Sync ---

// configDir returns the path to ~/.agentdesk/
func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".agentdesk")
}

// migrateDataDir performs a one-time migration from ~/.forge/ to ~/.agentdesk/
// for users upgrading from the previous branded version. It's a rename, not a
// copy, so it leaves nothing behind. Safe to call on every startup: it no-ops
// unless the old dir exists and the new one does not.
func migrateDataDir() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	oldDir := filepath.Join(home, ".forge")
	newDir := filepath.Join(home, ".agentdesk")

	oldInfo, err := os.Stat(oldDir)
	if err != nil || !oldInfo.IsDir() {
		return
	}
	if _, err := os.Stat(newDir); err == nil {
		// New dir already exists — user has already migrated or installed fresh.
		return
	}
	if err := os.Rename(oldDir, newDir); err != nil {
		log.Printf("data dir migration failed (will retry next launch): %v", err)
		return
	}
	log.Printf("migrated data directory: %s -> %s", oldDir, newDir)
}

// startTeamSyncIfConnected checks for an existing team connection and resumes sync.
func (a *App) startTeamSyncIfConnected() {
	configDir := configDir()
	if configDir == "" {
		return
	}
	state := team.LoadState(configDir)
	if !state.Connected || state.ServerURL == "" || state.Token == "" {
		return
	}
	client := team.NewClient(state.ServerURL, state.Token)
	a.teamClient = client
	ts := team.NewTeamSync(client, configDir)
	a.teamSync = ts
	ts.Start()
	log.Printf("team-sync: resumed connection to %s (%s)", state.ServerURL, state.OrgName)
}

// ConnectTeam validates a token against the team server, saves the config, and starts sync.
func (a *App) ConnectTeam(serverURL, token string) (team.TeamState, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	client := team.NewClient(serverURL, token)

	// Validate the token by calling GetMe
	user, err := client.GetMe()
	if err != nil {
		return team.TeamState{}, fmt.Errorf("connect failed: %w", err)
	}

	configDir := configDir()
	if configDir == "" {
		return team.TeamState{}, fmt.Errorf("cannot determine config directory")
	}

	// Stop existing team sync if running
	if a.teamSync != nil {
		a.teamSync.Stop()
		a.teamSync = nil
		a.teamClient = nil
	}

	// Save state
	state := team.LoadState(configDir)
	state.ServerURL = serverURL
	state.Token = token
	// /auth/me returns the member record; use email as org display name
	// since the endpoint doesn't include the org's name.
	orgName := user.Email
	if user.Name != nil && *user.Name != "" {
		orgName = *user.Name + "'s Org"
	}
	state.OrgName = orgName
	state.OrgID = user.OrgID
	state.Connected = true
	if state.ManagedFiles == nil {
		state.ManagedFiles = map[string]team.ManagedFile{}
	}

	if err := team.SaveState(configDir, state); err != nil {
		return team.TeamState{}, fmt.Errorf("save config: %w", err)
	}

	// Start sync
	a.teamClient = client
	ts := team.NewTeamSync(client, configDir)
	a.teamSync = ts
	ts.Start()

	log.Printf("team-sync: connected to %s (org: %s)", serverURL, orgName)

	// Emit event for frontend
	runtime.EventsEmit(a.ctx, "team-connected", map[string]string{
		"org": orgName,
	})

	return state, nil
}

// DisconnectTeam stops team sync and clears the config.
func (a *App) DisconnectTeam() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.teamSync != nil {
		a.teamSync.Stop()
		a.teamSync = nil
	}
	a.teamClient = nil

	configDir := configDir()
	if configDir == "" {
		return fmt.Errorf("cannot determine config directory")
	}

	// Clear connection info but preserve managed files list
	state := team.LoadState(configDir)
	state.Connected = false
	state.ServerURL = ""
	state.Token = ""
	state.OrgName = ""
	state.OrgID = ""
	state.LastSync = time.Time{}
	_ = team.SaveState(configDir, state)

	log.Println("team-sync: disconnected")
	runtime.EventsEmit(a.ctx, "team-disconnected", nil)

	return nil
}

// GetTeamState returns the current team connection state.
func (a *App) GetTeamState() team.TeamState {
	configDir := configDir()
	if configDir == "" {
		return team.TeamState{ManagedFiles: map[string]team.ManagedFile{}}
	}
	state := team.LoadState(configDir)
	// Strip token from the returned state for security
	state.Token = ""
	return state
}

// TeamSyncNow triggers an immediate sync cycle.
func (a *App) TeamSyncNow() error {
	if a.teamSync == nil {
		return fmt.Errorf("team sync not running")
	}
	return a.teamSync.SyncNow()
}

// GetTeamManagedPaths returns the absolute paths of all team-managed files.
func (a *App) GetTeamManagedPaths() []string {
	configDir := configDir()
	if configDir == "" {
		return nil
	}
	state := team.LoadState(configDir)
	if !state.Connected {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	paths := make([]string, 0, len(state.ManagedFiles))
	for relPath := range state.ManagedFiles {
		paths = append(paths, filepath.Join(home, relPath))
	}
	return paths
}

// IsTeamManaged returns true if the given file path is managed by the team.
func (a *App) IsTeamManaged(path string) bool {
	configDir := configDir()
	if configDir == "" {
		return false
	}
	state := team.LoadState(configDir)
	if !state.Connected {
		return false
	}
	// Check by absolute path — convert managed file keys to absolute paths
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	for relPath := range state.ManagedFiles {
		absPath := filepath.Join(home, relPath)
		if absPath == path {
			return true
		}
	}
	return false
}

// Version is set at build time via ldflags:
//
//	go build -ldflags "-X main.Version=v0.1.2-beta"
var Version = "dev"

// GetVersion returns the app version string.
func (a *App) GetVersion() string {
	return Version
}

// --- Window Visibility (Close to Background) ---

// Hide hides the application window. On Linux/Windows this is used to
// implement "close to background" behaviour.
func (a *App) Hide() {
	runtime.WindowHide(a.ctx)
}

// Show brings the application window back to the foreground.
func (a *App) Show() {
	runtime.WindowShow(a.ctx)
}

// --- Auto-Update Checker ---

// UpdateInfo contains information about available updates.
type UpdateInfo struct {
	Available      bool   `json:"available"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	DownloadURL    string `json:"downloadURL"`
}

// CheckForUpdate checks the GitHub releases API for a newer version of Forge.
// It respects a once-per-day throttle stored in settings. Returns an UpdateInfo
// indicating whether an update is available.
func (a *App) CheckForUpdate() UpdateInfo {
	current := Version
	noUpdate := UpdateInfo{
		Available:      false,
		CurrentVersion: current,
	}

	// Don't check if running a dev build
	if current == "dev" || current == "" {
		return noUpdate
	}

	// Throttle: at most once per day
	settings := skills.LoadSettings()
	now := time.Now().Unix()
	if settings.LastUpdateCheck > 0 && (now-settings.LastUpdateCheck) < 86400 {
		return noUpdate
	}

	// Update the last check timestamp
	settings.LastUpdateCheck = now
	_ = skills.SaveSettings(settings)

	info, err := fetchLatestRelease()
	if err != nil {
		log.Printf("update-check: %v", err)
		return noUpdate
	}

	latestTag := strings.TrimPrefix(info.TagName, "v")
	currentClean := strings.TrimPrefix(current, "v")

	if latestTag == "" || latestTag == currentClean {
		return noUpdate
	}

	// Simple string comparison; semver-aware comparison would be better,
	// but for most release patterns (v0.4.0 < v0.5.0) this works fine.
	if compareVersions(currentClean, latestTag) >= 0 {
		return noUpdate
	}

	return UpdateInfo{
		Available:      true,
		CurrentVersion: current,
		LatestVersion:  info.TagName,
		DownloadURL:    info.HTMLURL,
	}
}

// OpenURL opens the given URL in the user's default browser.
func (a *App) OpenURL(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
}

// githubRelease is the subset of the GitHub release API response we need.
type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// fetchLatestRelease calls the GitHub API and returns the latest release info.
func fetchLatestRelease() (githubRelease, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://api.github.com/repos/warunacds/Forge/releases/latest", nil)
	if err != nil {
		return githubRelease{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "AgentDesk/"+Version)

	resp, err := client.Do(req)
	if err != nil {
		return githubRelease{}, fmt.Errorf("fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return githubRelease{}, fmt.Errorf("github API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return githubRelease{}, fmt.Errorf("read response: %w", err)
	}

	var release githubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return githubRelease{}, fmt.Errorf("parse response: %w", err)
	}
	return release, nil
}

// compareVersions compares two version strings (e.g. "0.4.0" vs "0.5.0").
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareVersions(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}

	for i := 0; i < maxLen; i++ {
		var aNum, bNum int
		if i < len(aParts) {
			fmt.Sscanf(aParts[i], "%d", &aNum)
		}
		if i < len(bParts) {
			fmt.Sscanf(bParts[i], "%d", &bNum)
		}
		if aNum < bNum {
			return -1
		}
		if aNum > bNum {
			return 1
		}
	}
	return 0
}
