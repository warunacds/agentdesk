package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"agentdesk/internal/skills"
	syncpkg "agentdesk/internal/sync"
)

// Version is set at build time via ldflags.
var Version = "dev"

// Request is the JSON structure received on stdin.
type Request struct {
	Cmd    string          `json:"cmd"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Response is the JSON structure sent on stdout.
type Response struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

// Global state
var (
	syncNode   *syncpkg.Node
	syncEngine *syncpkg.Engine
	extraDirs  []string
)

func main() {
	log.SetOutput(os.Stderr) // logs go to stderr, JSON responses to stdout

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			writeError("invalid JSON: " + err.Error())
			continue
		}

		handleCommand(req)
	}
}

func handleCommand(req Request) {
	switch req.Cmd {

	// --- Skills ---
	case "scan":
		s := loadSettings()
		var extra []string
		for _, paths := range s.CustomPaths {
			extra = append(extra, paths...)
		}
		result := skills.Scan(extra)
		writeOK(result)

	case "save":
		var p struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		if err := os.WriteFile(p.Path, []byte(p.Content), 0644); err != nil {
			writeError(err.Error())
			return
		}
		writeOK(nil)

	case "create":
		var p struct {
			Path string          `json:"path"`
			Tool skills.ToolType `json:"tool"`
			Name string          `json:"name"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		tmpl := skills.GenerateTemplate(p.Tool, p.Name)
		if err := os.MkdirAll(os.ExpandEnv("$HOME"), 0755); err != nil {
			// ignore
		}
		dir := ""
		if len(p.Path) > 0 {
			dir = p.Path[:len(p.Path)-len(os.Args[0])]
		}
		_ = os.MkdirAll(dir, 0755)
		if err := os.WriteFile(p.Path, []byte(tmpl), 0644); err != nil {
			writeError(err.Error())
			return
		}
		writeOK(map[string]interface{}{
			"name": p.Name,
			"path": p.Path,
			"tool": p.Tool,
		})

	case "delete":
		var p struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		if err := os.Remove(p.Path); err != nil {
			writeError(err.Error())
			return
		}
		writeOK(nil)

	case "skill-disable":
		var p struct {
			Path string          `json:"path"`
			Tool skills.ToolType `json:"tool"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		if err := skills.DisableSkill(p.Path, p.Tool); err != nil {
			writeError(err.Error())
			return
		}
		writeOK(nil)

	case "skill-enable":
		var p struct {
			OriginalPath string `json:"originalPath"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		if err := skills.EnableSkill(p.OriginalPath); err != nil {
			writeError(err.Error())
			return
		}
		writeOK(nil)

	case "disabled-skills-list":
		writeOK(skills.ListDisabledSkills())

	case "skill-disable-for-tool":
		var p struct {
			Tool skills.ToolType `json:"tool"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		// Scan current skills and disable each for the given tool
		s := loadSettings()
		var extra []string
		for _, paths := range s.CustomPaths {
			extra = append(extra, paths...)
		}
		current := skills.Scan(extra)
		count := 0
		for _, sk := range current {
			if sk.Tool != p.Tool {
				continue
			}
			if skills.IsDisabled(sk.Path) {
				continue
			}
			if err := skills.DisableSkill(sk.Path, p.Tool); err != nil {
				writeError(err.Error())
				return
			}
			count++
		}
		writeOK(map[string]int{"count": count})

	case "skill-enable-all-for-tool":
		var p struct {
			Tool skills.ToolType `json:"tool"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		list := skills.ListDisabledSkills()
		count := 0
		for _, e := range list {
			if e.Tool != p.Tool {
				continue
			}
			if err := skills.EnableSkill(e.OriginalPath); err != nil {
				writeError(err.Error())
				return
			}
			count++
		}
		writeOK(map[string]int{"count": count})

	case "project-list":
		store := skills.LoadProjectStore()
		writeOK(store.Projects)

	case "project-add":
		var p struct {
			Path string `json:"path"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		project, err := skills.AddProject(p.Path, p.Name)
		if err != nil {
			writeError(err.Error())
			return
		}
		writeOK(project)

	case "project-remove":
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		if err := skills.RemoveProject(p.ID); err != nil {
			writeError(err.Error())
			return
		}
		writeOK(nil)

	case "project-rename":
		var p struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		if err := skills.RenameProject(p.ID, p.Name); err != nil {
			writeError(err.Error())
			return
		}
		writeOK(nil)

	case "project-scan-skills":
		var p struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		writeOK(skills.ScanProjectSkills(p.Path))

	case "project-override-set":
		var p struct {
			ProjectPath string `json:"projectPath"`
			SkillPath   string `json:"skillPath"`
			Disabled    bool   `json:"disabled"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		if err := skills.SetProjectOverride(p.ProjectPath, p.SkillPath, p.Disabled); err != nil {
			writeError(err.Error())
			return
		}
		writeOK(nil)

	case "project-overrides-get":
		var p struct {
			ProjectPath string `json:"projectPath"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		writeOK(map[string]interface{}{
			"disabledSkills": skills.GetProjectOverrides(p.ProjectPath),
		})

	case "read-file":
		var p struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		home, err := os.UserHomeDir()
		if err != nil {
			writeError("could not resolve home: " + err.Error())
			return
		}
		absPath, err := filepath.Abs(p.Path)
		if err != nil {
			writeError("invalid path: " + err.Error())
			return
		}
		absHome, _ := filepath.Abs(home)
		if !strings.HasPrefix(absPath, absHome+string(os.PathSeparator)) && absPath != absHome {
			writeError("path is outside user home")
			return
		}
		content, err := os.ReadFile(absPath)
		if err != nil {
			writeError(err.Error())
			return
		}
		lang := classifyLanguage(absPath)
		writeOK(map[string]interface{}{
			"content":  string(content),
			"size":     int64(len(content)),
			"language": lang,
		})

	case "update-metadata":
		var p struct {
			Path     string            `json:"path"`
			Metadata map[string]string `json:"metadata"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		content, err := os.ReadFile(p.Path)
		if err != nil {
			writeError(err.Error())
			return
		}
		updated := skills.ReplaceFrontmatter(string(content), p.Metadata)
		if err := os.WriteFile(p.Path, []byte(updated), 0644); err != nil {
			writeError(err.Error())
			return
		}
		writeOK(nil)

	// --- Tools ---
	case "known-tools":
		writeOK(skills.KnownTools)

	case "default-filename":
		var p struct {
			Tool skills.ToolType `json:"tool"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		writeOK(map[string]string{
			"filename": skills.DefaultFilename(p.Tool),
			"subdir":   skills.DefaultSubdir(p.Tool),
		})

	// --- Collections ---
	case "collections-list":
		store := skills.LoadCollections()
		writeOK(store.Collections)

	case "collection-create":
		var p struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		store := skills.LoadCollections()
		c := skills.Collection{
			ID:       fmt.Sprintf("%d", len(store.Collections)+1),
			Name:     p.Name,
			Created:  skills.NowUnix(),
			Modified: skills.NowUnix(),
		}
		store.Collections = append(store.Collections, c)
		if err := skills.SaveCollections(store); err != nil {
			writeError(err.Error())
			return
		}
		writeOK(c)

	case "collection-rename":
		var p struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		store := skills.LoadCollections()
		for i, c := range store.Collections {
			if c.ID == p.ID {
				store.Collections[i].Name = p.Name
				store.Collections[i].Modified = skills.NowUnix()
				break
			}
		}
		if err := skills.SaveCollections(store); err != nil {
			writeError(err.Error())
			return
		}
		writeOK(nil)

	case "collection-delete":
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		store := skills.LoadCollections()
		filtered := store.Collections[:0]
		for _, c := range store.Collections {
			if c.ID != p.ID {
				filtered = append(filtered, c)
			}
		}
		store.Collections = filtered
		if err := skills.SaveCollections(store); err != nil {
			writeError(err.Error())
			return
		}
		writeOK(nil)

	case "collection-add":
		var p struct {
			ID        string `json:"id"`
			SkillPath string `json:"skillPath"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		store := skills.LoadCollections()
		for i, c := range store.Collections {
			if c.ID == p.ID {
				store.Collections[i].SkillPaths = append(store.Collections[i].SkillPaths, p.SkillPath)
				store.Collections[i].Modified = skills.NowUnix()
				break
			}
		}
		if err := skills.SaveCollections(store); err != nil {
			writeError(err.Error())
			return
		}
		writeOK(nil)

	case "collection-remove":
		var p struct {
			ID        string `json:"id"`
			SkillPath string `json:"skillPath"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		store := skills.LoadCollections()
		for i, c := range store.Collections {
			if c.ID == p.ID {
				filtered := c.SkillPaths[:0]
				for _, sp := range c.SkillPaths {
					if sp != p.SkillPath {
						filtered = append(filtered, sp)
					}
				}
				store.Collections[i].SkillPaths = filtered
				store.Collections[i].Modified = skills.NowUnix()
				break
			}
		}
		if err := skills.SaveCollections(store); err != nil {
			writeError(err.Error())
			return
		}
		writeOK(nil)

	// --- Settings ---
	case "settings-get":
		writeOK(skills.LoadSettings())

	case "settings-save":
		var s skills.Settings
		if err := json.Unmarshal(req.Params, &s); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		if err := skills.SaveSettings(s); err != nil {
			writeError(err.Error())
			return
		}
		writeOK(nil)

	case "default-scan-paths":
		writeOK(skills.DefaultScanPaths())

	// --- Marketplace ---
	case "marketplace-sources":
		writeOK(skills.DefaultSources)

	case "marketplace-browse":
		var p struct {
			SourceID string `json:"sourceId"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		for _, src := range skills.DefaultSources {
			if src.ID == p.SourceID {
				result, err := skills.FetchMarketplaceSkills(src)
				if err != nil {
					writeError(err.Error())
					return
				}
				writeOK(result)
				return
			}
		}
		writeError("source not found")

	case "marketplace-preview":
		var p struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		content, err := skills.FetchSkillContent(p.URL)
		if err != nil {
			writeError(err.Error())
			return
		}
		writeOK(map[string]string{"content": content})

	case "marketplace-install":
		var p struct {
			URL      string          `json:"url"`
			Tool     skills.ToolType `json:"tool"`
			Filename string          `json:"filename"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		content, err := skills.FetchSkillContent(p.URL)
		if err != nil {
			writeError(err.Error())
			return
		}
		path, err := skills.InstallSkill(content, p.Tool, p.Filename)
		if err != nil {
			writeError(err.Error())
			return
		}
		writeOK(map[string]string{"path": path})

	// --- MCP ---
	case "mcp-configs":
		writeOK(skills.LoadAllMCPConfigs())

	// --- Plugins ---
	case "plugins":
		writeOK(skills.LoadAllPlugins())

	// --- Sync ---
	case "sync-status":
		if syncEngine == nil {
			writeOK(map[string]interface{}{"running": false})
			return
		}
		writeOK(syncEngine.GetSyncStatus())

	case "sync-peers":
		if syncEngine == nil {
			writeOK([]interface{}{})
			return
		}
		writeOK(syncEngine.GetPairedPeers())

	case "sync-add-peer":
		var p struct {
			PeerID string `json:"peerId"`
			Name   string `json:"name"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		if syncEngine == nil {
			writeError("sync not running")
			return
		}
		if err := syncpkg.AddPeerByString(syncEngine, syncNode, p.PeerID, p.Name); err != nil {
			writeError(err.Error())
			return
		}
		writeOK(nil)

	case "sync-remove-peer":
		var p struct {
			PeerID string `json:"peerId"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		if syncEngine == nil {
			writeError("sync not running")
			return
		}
		if err := syncEngine.RemovePeer(p.PeerID); err != nil {
			writeError(err.Error())
			return
		}
		writeOK(nil)

	case "device-id":
		if syncNode == nil {
			writeOK(map[string]string{"humanId": "", "fullId": ""})
			return
		}
		writeOK(map[string]string{
			"humanId": syncNode.HumanID(),
			"fullId":  syncNode.PeerID().String(),
		})

	// --- Sync Manifest ---
	case "sync-manifest":
		if syncEngine == nil {
			writeError("sync not running")
			return
		}
		writeOK(syncEngine.GetManifest())

	case "sync-add":
		var p struct {
			Paths []string `json:"paths"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		if syncEngine == nil {
			writeError("sync not running")
			return
		}
		if err := syncEngine.AddToManifest(p.Paths); err != nil {
			writeError(err.Error())
			return
		}
		writeOK(syncEngine.GetManifest())

	case "sync-remove":
		var p struct {
			Paths []string `json:"paths"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		if syncEngine == nil {
			writeError("sync not running")
			return
		}
		if err := syncEngine.RemoveFromManifest(p.Paths); err != nil {
			writeError(err.Error())
			return
		}
		writeOK(syncEngine.GetManifest())

	case "sync-file":
		var p struct {
			Paths []string `json:"paths"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		if syncEngine == nil {
			writeError("sync not running")
			return
		}
		if err := syncEngine.SyncFiles(p.Paths); err != nil {
			writeError(err.Error())
			return
		}
		writeOK(nil)

	case "sync-conflicts":
		if syncEngine == nil {
			writeOK([]interface{}{})
			return
		}
		writeOK(syncEngine.GetConflicts())

	case "sync-resolve":
		var p struct {
			Path       string `json:"path"`
			Resolution string `json:"resolution"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		if syncEngine == nil {
			writeError("sync not running")
			return
		}
		if err := syncEngine.ResolveConflict(p.Path, p.Resolution); err != nil {
			writeError(err.Error())
			return
		}
		writeOK(nil)

	// --- Recently Opened ---
	case "recent":
		writeOK(skills.LoadRecent())

	case "recent-add":
		var p struct {
			Path string `json:"path"`
			Name string `json:"name"`
			Tool string `json:"tool"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		skills.AddRecent(p.Path, p.Name, p.Tool)
		writeOK(nil)

	case "recent-clear":
		if err := skills.ClearRecent(); err != nil {
			writeError(err.Error())
			return
		}
		writeOK(nil)

	// --- Duplicate ---
	case "duplicate":
		var p struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		newSkill, err := duplicateSkill(p.Path)
		if err != nil {
			writeError(err.Error())
			return
		}
		writeOK(newSkill)

	// --- Search ---
	case "search":
		var p struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		s := loadSettings()
		var dirs []string
		for _, paths := range s.CustomPaths {
			dirs = append(dirs, paths...)
		}
		writeOK(skills.SearchFiles(p.Query, dirs))

	// --- Validation ---
	case "validate":
		s := loadSettings()
		var extra []string
		for _, paths := range s.CustomPaths {
			extra = append(extra, paths...)
		}
		allSkills := skills.Scan(extra)
		writeOK(skills.ValidateAll(allSkills))

	// --- Templates ---
	case "templates":
		writeOK(skills.BuiltinTemplates)

	case "create-from-template":
		var p struct {
			TemplateID string `json:"templateId"`
			Name       string `json:"name"`
			Path       string `json:"path"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError("invalid params: " + err.Error())
			return
		}
		result, err := createFromTemplate(p.TemplateID, p.Name, p.Path)
		if err != nil {
			writeError(err.Error())
			return
		}
		writeOK(result)

	// --- Version ---
	case "version":
		writeOK(map[string]string{"version": Version})

	// --- Watch (streaming) ---
	case "watch":
		runWatcher()

	default:
		writeError("unknown command: " + req.Cmd)
	}
}

// --- Sync helpers ---

func startSync() {
	node, err := syncpkg.NewNode()
	if err != nil {
		log.Printf("sync: failed to create node: %v", err)
		return
	}
	syncNode = node
	engine := syncpkg.NewEngine(node)
	syncEngine = engine
	node.SetEngine(engine)
	node.RegisterProtocol()
	node.SetOnDiscovery(func(evt syncpkg.DiscoveryEvent) {
		// Emit as a streaming event on stdout
		writeEvent("sync-discovery", map[string]string{
			"peerId": evt.PeerID,
			"event":  evt.Event,
		})
	})
	if err := node.StartDiscovery(); err != nil {
		log.Printf("sync: discovery error: %v", err)
	}
	engine.Start()
	log.Println("sync: started")
}

func stopSync() {
	if syncEngine != nil {
		syncEngine.Stop()
		syncEngine = nil
	}
	if syncNode != nil {
		syncNode.Stop()
		syncNode = nil
	}
	log.Println("sync: stopped")
}

func runWatcher() {
	watcher, err := skills.NewWatcher(func(path, op string) {
		writeEvent("file-changed", map[string]string{
			"path": path,
			"op":   op,
		})
		if syncEngine != nil {
			syncEngine.OnLocalFileChange(path)
		}
	})
	if err != nil {
		writeError("watcher: " + err.Error())
		return
	}
	watcher.Start()

	// Watch known directories
	home, _ := os.UserHomeDir()
	dirs := []string{
		home + "/.claude/agents",
		home + "/.claude/commands",
		home + "/.gemini/agents",
		home + "/.kiro/steering",
		home + "/.kiro/agents",
		home + "/.amp/skills",
		home + "/.cursor/rules",
		home + "/.codex/agents",
	}
	for _, d := range dirs {
		_ = watcher.Add(d)
	}

	writeOK(map[string]string{"status": "watching"})
	// Block forever — watcher runs in background goroutine
	select {}
}

// --- Output helpers ---

func writeOK(data interface{}) {
	resp := Response{OK: true, Data: data}
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
}

func writeError(msg string) {
	resp := Response{OK: false, Error: msg}
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
}

func writeEvent(event string, data interface{}) {
	evt := map[string]interface{}{
		"event": event,
		"data":  data,
	}
	b, _ := json.Marshal(evt)
	fmt.Println(string(b))
}

func loadSettings() skills.Settings {
	return skills.LoadSettings()
}

// duplicateSkill reads the file at path, writes a copy with a "-copy" suffix,
// and returns a map describing the new file.
func duplicateSkill(path string) (map[string]interface{}, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read source: %w", err)
	}

	newPath := deriveCopyPath(path)

	dir := filepath.Dir(newPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}

	if err := os.WriteFile(newPath, content, 0644); err != nil {
		return nil, fmt.Errorf("write copy: %w", err)
	}

	result := skills.Classify(newPath)
	fm, _ := skills.ParseFrontmatter(string(content))
	name := fm["name"]
	if name == "" {
		base := filepath.Base(newPath)
		name = strings.TrimSuffix(base, filepath.Ext(base))
	}

	return map[string]interface{}{
		"name": name,
		"path": newPath,
		"tool": result.Tool,
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

// createFromTemplate creates a new skill file from a built-in template.
func createFromTemplate(templateID, name, path string) (map[string]interface{}, error) {
	tmpl := skills.FindTemplate(templateID)
	if tmpl == nil {
		return nil, fmt.Errorf("template not found: %s", templateID)
	}

	content := tmpl.Content

	// Replace the name in frontmatter if the content has one
	if name != "" && strings.Contains(content, "---") {
		fm, body := skills.ParseFrontmatter(content)
		fm["name"] = name
		content = skills.ReplaceFrontmatter(body, fm)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}

	fm, _ := skills.ParseFrontmatter(content)
	displayName := fm["name"]
	if displayName == "" {
		displayName = name
	}

	return map[string]interface{}{
		"name": displayName,
		"path": path,
		"tool": tmpl.Tool,
	}, nil
}

func classifyLanguage(path string) string {
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

