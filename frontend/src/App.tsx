import { useState, useEffect, useCallback, useRef, CSSProperties } from "react";
import {
  GetSkills, GetKnownTools, AddDirectory, SelectDirectory,
  GetCollections, CreateCollection, RenameCollection, DeleteCollection,
  AddToCollection, RemoveFromCollection,
  GetSettings, SaveSettings as SaveSettingsBackend,
  // GetSyncStatus will be available after `wails build` generates bindings
  GetSyncStatus,
  GetMCPConfigs,
  GetPlugins,
  // Team sync bindings — available after `wails build`
  GetTeamState,
  ConnectTeam,
  DisconnectTeam,
  TeamSyncNow,
  GetTeamManagedPaths,
  // Selective sync bindings
  GetSyncManifest,
  AddToSyncManifest,
  RemoveFromSyncManifest,
  SyncFiles,
  GetSyncConflicts,
  ResolveSyncConflict,
  // New bindings
  GetRecent,
  AddRecent,
  ClearRecent,
  DuplicateSkill,
  SearchSkills,
  ValidateSkills,
  GetTemplates,
  DeleteSkill,
  // Auto-update bindings
  CheckForUpdate,
  OpenURL,
  // Disable / enable skill bindings
  DisableSkill,
  EnableSkill,
  ListDisabledSkills,
  DisableSkillsForTool,
  EnableAllDisabledForTool,
  // Project bindings
  ListProjects,
  AddProject,
  RemoveProject,
  RenameProject,
  ScanProjectSkills,
  SetProjectSkillOverride,
  GetProjectOverrides,
  SelectProjectDirectory,
} from "../wailsjs/go/main/App";
import { EventsOn, EventsOff } from "../wailsjs/runtime/runtime";
import { Skill, ToolMeta, ToolType, Collection, Settings, SyncStatus, SyncManifest, SyncConflict, MCPToolConfig, PluginToolConfig, TeamState, SearchResult, ValidationWarning, SkillTemplate, RecentFile, UpdateInfo, Project } from "./types";
import { Sidebar } from "./components/Sidebar";
import { SkillEditor } from "./components/SkillEditor";
import { NewSkillModal } from "./components/NewSkillModal";
import { SettingsPanel } from "./components/SettingsPanel";
import { MarketplacePanel } from "./components/MarketplacePanel";
import { MCPPanel } from "./components/MCPPanel";
import { PluginsPanel } from "./components/PluginsPanel";
import { Toast } from "./components/Toast";
import { SyncConflictDialog, SyncConflictBanner } from "./components/SyncConflictDialog";

interface ToastState {
  message: string;
  undoAction: () => void;
}

interface SyncToastState {
  message: string;
  id: number;
}

export default function App() {
  const [skills, setSkills]                   = useState<Skill[]>([]);
  const [tools, setTools]                     = useState<ToolMeta[]>([]);
  const [collections, setCollections]         = useState<Collection[]>([]);
  const [activeId, setActiveId]               = useState<string | null>(null);
  const [activeCollection, setActiveCollection] = useState<string | null>(null);
  const [loading, setLoading]                 = useState(true);
  const [showNew, setShowNew]                 = useState(false);
  const [showSettings, setShowSettings]       = useState(false);
  const [showMarketplace, setShowMarketplace] = useState(false);
  const [showMCP, setShowMCP]                 = useState(false);
  const [showPlugins, setShowPlugins]         = useState(false);
  const [mcpConfigs, setMcpConfigs]           = useState<MCPToolConfig[]>([]);
  const [pluginConfigs, setPluginConfigs]     = useState<PluginToolConfig[]>([]);
  const [settings, setSettings]               = useState<Settings>({ customPaths: {}, theme: "dark", scanOnStartup: true, runInBackground: false, lastUpdateCheck: 0, dismissedVersion: "" });
  const [updateInfo, setUpdateInfo]           = useState<UpdateInfo | null>(null);
  const [syncStatus, setSyncStatus]           = useState<SyncStatus | null>(null);
  const [teamState, setTeamState]             = useState<TeamState | null>(null);
  const [teamError, setTeamError]             = useState<string | null>(null);
  const [teamManagedPaths, setTeamManagedPaths] = useState<Set<string>>(new Set());
  const [syncManifest, setSyncManifest]         = useState<SyncManifest | null>(null);
  const [syncConflicts, setSyncConflicts]       = useState<SyncConflict[]>([]);
  const [syncingPaths, setSyncingPaths]         = useState<Set<string>>(new Set());
  const [flashPaths, setFlashPaths]             = useState<Set<string>>(new Set());
  const [showConflictDialog, setShowConflictDialog] = useState(false);
  const [syncToast, setSyncToast]               = useState<SyncToastState | null>(null);
  const [toast, setToast]                     = useState<ToastState | null>(null);
  const [recentFiles, setRecentFiles]           = useState<RecentFile[]>([]);
  const [validationWarnings, setValidationWarnings] = useState<Record<string, ValidationWarning[]>>({});
  const [searchResults, setSearchResults]       = useState<SearchResult[]>([]);
  const [selectedPaths, setSelectedPaths]       = useState<Set<string>>(new Set());
  const [templates, setTemplates]               = useState<SkillTemplate[]>([]);
  const [disabledPaths, setDisabledPaths]       = useState<Set<string>>(new Set());
  const [projects, setProjects]                 = useState<Project[]>([]);
  const [projectSkillsByPath, setProjectSkillsByPath] = useState<Record<string, Skill[]>>({});
  const [projectOverridesByPath, setProjectOverridesByPath] = useState<Record<string, Set<string>>>({});
  const dismissToast = useCallback(() => setToast(null), []);

  const loadSkills = useCallback(async () => {
    setLoading(true);
    try {
      const [s, t, c, st] = await Promise.all([GetSkills(), GetKnownTools(), GetCollections(), GetSettings()]);
      setSkills((s ?? []) as unknown as Skill[]);
      setTools((t ?? []) as unknown as ToolMeta[]);
      setCollections((c ?? []) as unknown as Collection[]);
      if (st) setSettings(st as unknown as Settings);
    } finally {
      setLoading(false);
    }
  }, []);

  const reloadCollections = useCallback(async () => {
    const c = await GetCollections();
    setCollections((c ?? []) as unknown as Collection[]);
  }, []);

  const loadDisabled = useCallback(async () => {
    try {
      const list = await ListDisabledSkills();
      setDisabledPaths(new Set((list ?? []).map((d) => d.originalPath)));
    } catch {
      setDisabledPaths(new Set());
    }
  }, []);

  const handleDisableSkill = useCallback(async (path: string, tool: ToolType) => {
    await DisableSkill(path, tool);
  }, []);

  const handleEnableSkill = useCallback(async (originalPath: string) => {
    await EnableSkill(originalPath);
  }, []);

  const handleDisableAllForTool = useCallback(async (tool: ToolType) => {
    await DisableSkillsForTool(tool);
  }, []);

  const handleEnableAllForTool = useCallback(async (tool: ToolType) => {
    await EnableAllDisabledForTool(tool);
  }, []);

  // --- Project handlers ---

  const loadProjects = useCallback(async () => {
    try {
      const list = await ListProjects();
      setProjects((list ?? []) as unknown as Project[]);
    } catch {
      setProjects([]);
    }
  }, []);

  const handleAddProject = useCallback(async () => {
    try {
      const path = await SelectProjectDirectory();
      if (!path) return;
      await AddProject(path, "");
      await loadProjects();
    } catch (err) {
      console.error("Failed to add project:", err);
    }
  }, [loadProjects]);

  const handleRemoveProject = useCallback(async (id: string) => {
    await RemoveProject(id);
    await loadProjects();
  }, [loadProjects]);

  const handleRenameProject = useCallback(async (id: string, name: string) => {
    await RenameProject(id, name);
    await loadProjects();
  }, [loadProjects]);

  const loadProjectSkills = useCallback(async (projectPath: string) => {
    try {
      const list = await ScanProjectSkills(projectPath);
      setProjectSkillsByPath((prev) => ({ ...prev, [projectPath]: (list ?? []) as unknown as Skill[] }));
      const overrides = await GetProjectOverrides(projectPath);
      setProjectOverridesByPath((prev) => ({ ...prev, [projectPath]: new Set(overrides ?? []) }));
    } catch {
      // Scan or overrides may fail
    }
  }, []);

  const handleToggleOverride = useCallback(async (projectPath: string, skillPath: string, disabled: boolean) => {
    await SetProjectSkillOverride(projectPath, skillPath, disabled);
    const overrides = await GetProjectOverrides(projectPath);
    setProjectOverridesByPath((prev) => ({ ...prev, [projectPath]: new Set(overrides ?? []) }));
  }, []);

  const loadSyncStatus = useCallback(async () => {
    try {
      const status = await GetSyncStatus();
      if (status) setSyncStatus(status as unknown as SyncStatus);
    } catch {
      // Sync methods may not exist yet until wails build generates bindings
    }
  }, []);

  const loadSyncManifest = useCallback(async () => {
    try {
      const manifest = await GetSyncManifest();
      if (manifest) setSyncManifest(manifest as unknown as SyncManifest);
    } catch {
      // Sync manifest methods may not exist yet until wails build generates bindings
    }
  }, []);

  const loadSyncConflicts = useCallback(async () => {
    try {
      const conflicts = await GetSyncConflicts();
      if (conflicts) setSyncConflicts((conflicts ?? []) as unknown as SyncConflict[]);
    } catch {
      // Sync conflict methods may not exist yet until wails build generates bindings
    }
  }, []);

  const loadMCPConfigs = useCallback(async () => {
    try {
      const configs = await GetMCPConfigs();
      if (configs) setMcpConfigs(configs as unknown as MCPToolConfig[]);
    } catch {
      // MCP methods may not exist yet until wails build generates bindings
    }
  }, []);

  const loadPluginConfigs = useCallback(async () => {
    try {
      const configs = await GetPlugins();
      if (configs) setPluginConfigs(configs as unknown as PluginToolConfig[]);
    } catch {
      // Plugin methods may not exist yet until wails build generates bindings
    }
  }, []);

  const loadRecent = useCallback(async () => {
    try {
      const store = await GetRecent();
      if (store && (store as any).files) setRecentFiles(((store as any).files ?? []) as RecentFile[]);
    } catch {
      // Recent methods may not exist yet until wails build generates bindings
    }
  }, []);

  const loadValidation = useCallback(async () => {
    try {
      const warnings = await ValidateSkills();
      if (warnings) setValidationWarnings(warnings as unknown as Record<string, ValidationWarning[]>);
    } catch {
      // Validation methods may not exist yet until wails build generates bindings
    }
  }, []);

  const loadTemplates = useCallback(async () => {
    try {
      const tmpls = await GetTemplates();
      if (tmpls) setTemplates((tmpls ?? []) as unknown as SkillTemplate[]);
    } catch {
      // Template methods may not exist yet until wails build generates bindings
    }
  }, []);

  // --- Team sync handlers ---

  const loadTeamState = useCallback(async () => {
    try {
      const state = await GetTeamState();
      if (state) {
        const ts = state as unknown as TeamState;
        setTeamState(ts);
        // Build a Set of absolute managed file paths for quick lookup
        if (ts.connected && ts.managed_files) {
          // managed_files keys are relative paths — resolved via GetTeamManagedPaths
        }
      }
    } catch {
      // Team methods may not exist yet until wails build generates bindings
    }
  }, []);

  const refreshTeamManagedPaths = useCallback(async () => {
    try {
      const paths = await GetTeamManagedPaths();
      setTeamManagedPaths(new Set(paths || []));
    } catch {
      setTeamManagedPaths(new Set());
    }
  }, []);

  const handleConnectTeam = useCallback(async (serverURL: string, token: string) => {
    setTeamError(null);
    try {
      const result = await ConnectTeam(serverURL, token);
      setTeamState(result as unknown as TeamState);
      setTeamError(null);
      // Reload skills after connecting to pick up newly synced files
      const s = await GetSkills();
      const newSkills = (s ?? []) as unknown as Skill[];
      setSkills(newSkills);
      await refreshTeamManagedPaths();
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      setTeamError(message);
    }
  }, [refreshTeamManagedPaths]);

  const handleDisconnectTeam = useCallback(async () => {
    try {
      await DisconnectTeam();
      setTeamState(null);
      setTeamManagedPaths(new Set());
      setTeamError(null);
    } catch {
      // Disconnect may fail
    }
  }, []);

  const handleTeamSyncNow = useCallback(async () => {
    try {
      await TeamSyncNow();
      await loadTeamState();
      // Reload skills to pick up changes
      const s = await GetSkills();
      const newSkills = (s ?? []) as unknown as Skill[];
      setSkills(newSkills);
      await refreshTeamManagedPaths();
    } catch {
      // Sync may fail
    }
  }, [loadTeamState, refreshTeamManagedPaths]);

  // --- Selective sync handlers ---

  // Convert absolute path to tilde-relative for the manifest
  const toRelPath = useCallback((absPath: string): string => {
    const home = absPath.match(/^(\/Users\/[^/]+|\/home\/[^/]+)/);
    if (home) return absPath.replace(home[1], "~");
    return absPath;
  }, []);

  const handleSyncFile = useCallback(async (absPath: string) => {
    const relPath = toRelPath(absPath);
    setSyncingPaths((prev) => new Set(prev).add(absPath));
    try {
      await AddToSyncManifest([relPath]);
      await SyncFiles([relPath]);
      await loadSyncManifest();
    } catch {
      // Sync may fail
    } finally {
      setSyncingPaths((prev) => {
        const next = new Set(prev);
        next.delete(absPath);
        return next;
      });
    }
  }, [toRelPath, loadSyncManifest]);

  const handleStopSyncFile = useCallback(async (absPath: string) => {
    const relPath = toRelPath(absPath);
    try {
      await RemoveFromSyncManifest([relPath]);
      await loadSyncManifest();
    } catch {
      // Sync may fail
    }
  }, [toRelPath, loadSyncManifest]);

  const handleResolveConflict = useCallback(async (path: string, resolution: string) => {
    try {
      await ResolveSyncConflict(path, resolution);
      await loadSyncConflicts();
    } catch {
      // Resolution may fail
    }
  }, [loadSyncConflicts]);

  // Flash a skill row briefly with green tint
  const flashSkillPath = useCallback((absPath: string) => {
    setFlashPaths((prev) => new Set(prev).add(absPath));
    setTimeout(() => {
      setFlashPaths((prev) => {
        const next = new Set(prev);
        next.delete(absPath);
        return next;
      });
    }, 1000);
  }, []);

  // Ref to access current skills in event handlers without adding them as dependencies
  const skillsRef = useRef(skills);
  skillsRef.current = skills;

  // Apply theme to document root whenever settings change
  useEffect(() => {
    document.documentElement.setAttribute('data-theme', settings?.theme || 'dark');
  }, [settings?.theme]);

  useEffect(() => {
    loadSkills().then(() => loadValidation());
    loadRecent();
    loadTemplates();
    loadDisabled();
    loadProjects();
    // Non-blocking auto-update check on startup
    CheckForUpdate().then((info) => {
      const update = info as unknown as UpdateInfo;
      if (update?.available) setUpdateInfo(update);
    }).catch(() => {});
    EventsOn("file-changed", () => { loadSkills().then(() => loadValidation()); });
    EventsOn("skills-changed", () => { loadSkills().then(() => loadValidation()); loadDisabled(); });
    return () => { EventsOff("file-changed"); EventsOff("skills-changed"); };
  }, [loadSkills, loadRecent, loadValidation, loadTemplates, loadDisabled, loadProjects]);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "r") { e.preventDefault(); loadSkills(); }
      if ((e.metaKey || e.ctrlKey) && e.key === "n") { e.preventDefault(); setShowNew(true); }
      if ((e.metaKey || e.ctrlKey) && e.key === ",") { e.preventDefault(); setShowSettings(true); setShowMarketplace(false); setShowMCP(false); setShowPlugins(false); }
      if ((e.metaKey || e.ctrlKey) && e.key === "m") { e.preventDefault(); setShowMCP(true); setShowSettings(false); setShowMarketplace(false); setShowPlugins(false); loadMCPConfigs(); }
      if ((e.metaKey || e.ctrlKey) && e.key === "b") { e.preventDefault(); setShowMarketplace(true); setShowSettings(false); setShowMCP(false); setShowPlugins(false); }
      if ((e.metaKey || e.ctrlKey) && e.key === "p") { e.preventDefault(); setShowPlugins(true); setShowSettings(false); setShowMarketplace(false); setShowMCP(false); loadPluginConfigs(); }
      if (e.key === "Escape") { e.preventDefault(); setShowSettings(false); setShowMarketplace(false); setShowMCP(false); setShowPlugins(false); }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [loadSkills, loadMCPConfigs, loadPluginConfigs]);

  // Load sync status, team state, sync manifest, and conflicts on startup; poll every 10s
  useEffect(() => {
    loadSyncStatus();
    loadTeamState();
    loadSyncManifest();
    loadSyncConflicts();
    const interval = setInterval(() => {
      loadSyncStatus();
      loadTeamState();
      loadSyncManifest();
      loadSyncConflicts();
    }, 10_000);
    return () => clearInterval(interval);
  }, [loadSyncStatus, loadTeamState, loadSyncManifest, loadSyncConflicts]);

  // Listen for sync-related events
  useEffect(() => {
    EventsOn("sync-discovery", () => loadSyncStatus());
    EventsOn("file-synced", (data: { path?: string; peerId?: string }) => {
      loadSyncStatus();
      loadSkills();
      loadSyncManifest();
      // Show sync toast notification
      if (data?.path) {
        const fileName = data.path.split("/").pop() || data.path;
        setSyncToast({ message: `\u2193 ${fileName} synced from peer`, id: Date.now() });
        // Find the absolute path for flash animation
        // The event path is tilde-relative; try to find a matching skill
        const home = data.path.replace(/^~/, "");
        const matchingSkill = skillsRef.current.find((s) => s.path.endsWith(home));
        if (matchingSkill) flashSkillPath(matchingSkill.path);
      }
    });
    EventsOn("sync-conflict", () => { loadSyncConflicts(); });
    EventsOn("sync-conflict-stored", () => { loadSyncConflicts(); });
    EventsOn("team-connected", () => { loadTeamState(); loadSkills(); });
    EventsOn("team-disconnected", () => { loadTeamState(); setTeamManagedPaths(new Set()); });
    return () => {
      EventsOff("sync-discovery");
      EventsOff("file-synced");
      EventsOff("sync-conflict");
      EventsOff("sync-conflict-stored");
      EventsOff("team-connected");
      EventsOff("team-disconnected");
    };
  }, [loadSyncStatus, loadSkills, loadTeamState, loadSyncManifest, loadSyncConflicts, flashSkillPath]);

  // Refresh team managed paths whenever skills list changes
  useEffect(() => {
    if (teamState?.connected && skills.length > 0) {
      refreshTeamManagedPaths();
    }
  }, [skills, teamState?.connected, refreshTeamManagedPaths]);

  const activeSkill = skills.find((s) => s.id === activeId) ?? null;

  const handleAddDir = async () => {
    const path = await SelectDirectory();
    if (path) { await AddDirectory(path); loadSkills(); }
  };

  // --- Collection handlers ---

  const handleCreateCollection = async (name: string) => {
    await CreateCollection(name);
    await reloadCollections();
  };

  const handleRenameCollection = async (id: string, name: string) => {
    await RenameCollection(id, name);
    await reloadCollections();
  };

  const handleDeleteCollection = async (id: string) => {
    // Save collection data before deleting for undo
    const collectionToDelete = collections.find((c) => c.id === id);
    const savedName = collectionToDelete?.name ?? "";
    const savedPaths = collectionToDelete?.skillPaths ? [...collectionToDelete.skillPaths] : [];

    await DeleteCollection(id);
    if (activeCollection === id) setActiveCollection(null);
    await reloadCollections();

    // Show undo toast
    if (savedName) {
      setToast({
        message: "Collection deleted",
        undoAction: async () => {
          await CreateCollection(savedName);
          // Re-fetch collections to get the new ID
          const updatedCollections = await GetCollections();
          const newColl = (updatedCollections ?? []) as unknown as Collection[];
          const recreated = newColl.find((c) => c.name === savedName);
          if (recreated) {
            for (const skillPath of savedPaths) {
              await AddToCollection(recreated.id, skillPath);
            }
          }
          await reloadCollections();
        },
      });
    }
  };

  const handleAddToCollection = async (collectionId: string, skillPath: string) => {
    await AddToCollection(collectionId, skillPath);
    await reloadCollections();
  };

  const handleRemoveFromCollection = async (collectionId: string, skillPath: string) => {
    await RemoveFromCollection(collectionId, skillPath);
    await reloadCollections();

    // Show undo toast
    setToast({
      message: "Removed from collection",
      undoAction: async () => {
        await AddToCollection(collectionId, skillPath);
        await reloadCollections();
      },
    });
  };

  // --- Recent, Search, Duplicate, Validation, Multi-select handlers ---

  const handleSelectSkill = useCallback((skill: Skill) => {
    setActiveId(skill.id);
    setShowSettings(false); setShowMarketplace(false); setShowMCP(false); setShowPlugins(false);
    setSelectedPaths(new Set());
    // Track recent
    AddRecent(skill.path, skill.name, skill.tool).catch(() => {});
    loadRecent();
  }, [loadRecent]);

  const handleClearRecent = useCallback(async () => {
    try {
      await ClearRecent();
      setRecentFiles([]);
    } catch {
      // May not exist yet
    }
  }, []);

  const handleDuplicate = useCallback(async (path: string) => {
    try {
      const newSkill = await DuplicateSkill(path);
      if (newSkill) {
        const fileName = (newSkill as unknown as Skill).name || path.split("/").pop() || "file";
        setToast({ message: `Duplicated as ${fileName}`, undoAction: () => {} });
        await loadSkills();
        await loadValidation();
      }
    } catch (err) {
      console.error("Duplicate failed:", err);
    }
  }, [loadSkills, loadValidation]);

  const handleSearchSkills = useCallback(async (query: string) => {
    try {
      const results = await SearchSkills(query);
      setSearchResults((results ?? []) as unknown as SearchResult[]);
    } catch {
      setSearchResults([]);
    }
  }, []);

  const handleSelectSearchResult = useCallback((result: SearchResult) => {
    const skill = skills.find((s) => s.path === result.path);
    if (skill) {
      setActiveId(skill.id);
      setShowSettings(false); setShowMarketplace(false); setShowMCP(false); setShowPlugins(false);
      setSelectedPaths(new Set());
    }
  }, [skills]);

  // Multi-select: Cmd+click toggle
  const handleToggleSelect = useCallback((path: string, _meta: boolean) => {
    setSelectedPaths((prev) => {
      const next = new Set(prev);
      if (next.has(path)) {
        next.delete(path);
      } else {
        next.add(path);
      }
      return next;
    });
  }, []);

  // Multi-select: Shift+click range select
  const lastClickedPathRef = useRef<string | null>(null);
  const handleRangeSelect = useCallback((path: string) => {
    const allPaths = skills.map((s) => s.path);
    const lastPath = lastClickedPathRef.current;
    if (!lastPath) {
      setSelectedPaths(new Set([path]));
      lastClickedPathRef.current = path;
      return;
    }
    const fromIdx = allPaths.indexOf(lastPath);
    const toIdx = allPaths.indexOf(path);
    if (fromIdx === -1 || toIdx === -1) {
      setSelectedPaths(new Set([path]));
      lastClickedPathRef.current = path;
      return;
    }
    const start = Math.min(fromIdx, toIdx);
    const end = Math.max(fromIdx, toIdx);
    const range = allPaths.slice(start, end + 1);
    setSelectedPaths((prev) => {
      const next = new Set(prev);
      range.forEach((p) => next.add(p));
      return next;
    });
  }, [skills]);

  // Track last clicked path for range select
  useEffect(() => {
    if (activeId) {
      const skill = skills.find((s) => s.id === activeId);
      if (skill) lastClickedPathRef.current = skill.path;
    }
  }, [activeId, skills]);

  // Bulk delete with confirmation
  const handleBulkDelete = useCallback(async () => {
    const count = selectedPaths.size;
    if (count === 0) return;
    const confirmed = window.confirm(`Delete ${count} skill${count !== 1 ? "s" : ""}? This cannot be undone.`);
    if (!confirmed) return;
    try {
      for (const path of selectedPaths) {
        await DeleteSkill(path);
      }
      setSelectedPaths(new Set());
      await loadSkills();
      await loadValidation();
    } catch (err) {
      console.error("Bulk delete failed:", err);
    }
  }, [selectedPaths, loadSkills, loadValidation]);

  // Bulk add to collection
  const handleBulkAddToCollection = useCallback(async (collectionId: string) => {
    try {
      for (const path of selectedPaths) {
        await AddToCollection(collectionId, path);
      }
      await reloadCollections();
      setSelectedPaths(new Set());
    } catch (err) {
      console.error("Bulk add to collection failed:", err);
    }
  }, [selectedPaths, reloadCollections]);

  // Bulk sync
  const handleBulkSync = useCallback(async () => {
    try {
      for (const path of selectedPaths) {
        await handleSyncFile(path);
      }
      setSelectedPaths(new Set());
    } catch (err) {
      console.error("Bulk sync failed:", err);
    }
  }, [selectedPaths, handleSyncFile]);

  // --- Settings handler ---

  const handleSaveSettings = async (updated: Settings) => {
    await SaveSettingsBackend(updated as any);
    setSettings(updated);
    // Re-scan skills since custom paths may have changed
    const s = await GetSkills();
    setSkills((s ?? []) as unknown as Skill[]);
  };

  // --- Title bar styles ---

  const titleBarStyle: CSSProperties & Record<string, unknown> = {
    height: "var(--titlebar-h)",
    "--wails-draggable": "drag",
    WebkitAppRegion: "drag",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    borderBottom: "1px solid var(--border)",
    background: "var(--bg)",
    flexShrink: 0,
    position: "relative",
    zIndex: 10,
  };

  const noDragStyle: CSSProperties & Record<string, unknown> = {
    "--wails-draggable": "no-drag",
    WebkitAppRegion: "no-drag",
    position: "absolute" as const,
    right: 12,
    display: "flex",
    gap: 6,
  };

  return (
    <>
      <div style={titleBarStyle}>
        <span style={{ fontFamily: "var(--font-mono)", fontSize: 12, fontWeight: 600, color: "var(--text-3)", letterSpacing: "0.1em", textTransform: "uppercase" }}>Agent Desk</span>
        <div style={noDragStyle}>
          <button onClick={loadSkills} style={{ background: "transparent", border: "1px solid var(--border)", borderRadius: "var(--radius)", color: "var(--text-2)", padding: "3px 8px", fontSize: 11, cursor: "pointer" }}>{"\u21ba"}</button>
        </div>
      </div>

      {/* Auto-update banner */}
      {updateInfo?.available && (
        <div style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          gap: 8,
          padding: "5px 16px",
          background: "var(--accent-dim)",
          borderBottom: "1px solid var(--border)",
          fontSize: 12,
          color: "var(--text-2)",
          fontFamily: "var(--font-ui)",
          flexShrink: 0,
        }}>
          <span>Agent Desk {updateInfo.latestVersion} is available</span>
          <span style={{ color: "var(--text-3)" }}>{"\u2014"}</span>
          <button
            onClick={() => { if (updateInfo.downloadURL) OpenURL(updateInfo.downloadURL); }}
            style={{
              background: "transparent",
              border: "none",
              color: "var(--accent)",
              cursor: "pointer",
              fontSize: 12,
              fontFamily: "var(--font-ui)",
              fontWeight: 600,
              textDecoration: "underline",
              padding: 0,
            }}
          >
            Download
          </button>
          <button
            onClick={() => setUpdateInfo(null)}
            style={{
              background: "transparent",
              border: "none",
              color: "var(--text-3)",
              cursor: "pointer",
              fontSize: 14,
              padding: "0 4px",
              lineHeight: 1,
              marginLeft: 4,
            }}
            title="Dismiss"
          >
            {"\u2715"}
          </button>
        </div>
      )}

      <div style={{ display: "flex", height: updateInfo?.available ? "calc(100vh - var(--titlebar-h) - 31px)" : "calc(100vh - var(--titlebar-h))" }}>
        <Sidebar
          skills={skills}
          tools={tools}
          collections={collections}
          activeId={activeId}
          activeCollection={activeCollection}
          syncStatus={syncStatus}
          teamManagedPaths={teamManagedPaths}
          syncManifest={syncManifest}
          syncConflicts={syncConflicts}
          syncingPaths={syncingPaths}
          flashPaths={flashPaths}
          recentFiles={recentFiles}
          validationWarnings={validationWarnings}
          selectedPaths={selectedPaths}
          onSelect={handleSelectSkill}
          onNew={() => setShowNew(true)}
          onAddDir={handleAddDir}
          onOpenSettings={() => { setShowSettings(true); setShowMarketplace(false); setShowMCP(false); setShowPlugins(false); }}
          onOpenMarketplace={() => { setShowMarketplace(true); setShowSettings(false); setShowMCP(false); setShowPlugins(false); }}
          onOpenMCP={() => { setShowMCP(true); setShowSettings(false); setShowMarketplace(false); setShowPlugins(false); loadMCPConfigs(); }}
          onOpenPlugins={() => { setShowPlugins(true); setShowSettings(false); setShowMarketplace(false); setShowMCP(false); loadPluginConfigs(); }}
          onSelectCollection={(id) => setActiveCollection(id)}
          onCreateCollection={handleCreateCollection}
          onRenameCollection={handleRenameCollection}
          onDeleteCollection={handleDeleteCollection}
          onAddToCollection={handleAddToCollection}
          onRemoveFromCollection={handleRemoveFromCollection}
          onSyncFile={handleSyncFile}
          onStopSyncFile={handleStopSyncFile}
          onDuplicate={handleDuplicate}
          onClearRecent={handleClearRecent}
          onSearchSkills={handleSearchSkills}
          searchResults={searchResults}
          onSelectSearchResult={handleSelectSearchResult}
          onToggleSelect={handleToggleSelect}
          onRangeSelect={handleRangeSelect}
          onBulkDelete={handleBulkDelete}
          onBulkAddToCollection={handleBulkAddToCollection}
          onBulkSync={handleBulkSync}
          disabledPaths={disabledPaths}
          onDisableSkill={handleDisableSkill}
          onEnableSkill={handleEnableSkill}
          onDisableAllForTool={handleDisableAllForTool}
          onEnableAllForTool={handleEnableAllForTool}
          projects={projects}
          projectSkillsByPath={projectSkillsByPath}
          projectOverridesByPath={projectOverridesByPath}
          onAddProject={handleAddProject}
          onRemoveProject={handleRemoveProject}
          onRenameProject={handleRenameProject}
          onLoadProjectSkills={loadProjectSkills}
          onToggleProjectOverride={handleToggleOverride}
          loading={loading}
        />
        <div style={{ flex: 1, display: "flex", flexDirection: "column", overflow: "hidden", background: "var(--bg)" }}>
          <SyncConflictBanner conflictCount={syncConflicts.length} onReview={() => setShowConflictDialog(true)} />
          {showPlugins
            ? <PluginsPanel configs={pluginConfigs} onClose={() => setShowPlugins(false)} onRefresh={loadPluginConfigs} />
            : showMCP
            ? <MCPPanel configs={mcpConfigs} onClose={() => setShowMCP(false)} onRefresh={loadMCPConfigs} />
            : showMarketplace
            ? <MarketplacePanel tools={tools} onClose={() => setShowMarketplace(false)} onInstalled={loadSkills} />
            : showSettings
            ? <SettingsPanel settings={settings} tools={tools} syncStatus={syncStatus} onSave={handleSaveSettings} onClose={() => setShowSettings(false)} onRefreshSync={loadSyncStatus} />
            : activeSkill
            ? <SkillEditor key={activeSkill.id + settings.theme} skill={activeSkill} tools={tools} theme={settings.theme || "dark"} isTeamManaged={teamManagedPaths.has(activeSkill.path)} validationWarnings={validationWarnings[activeSkill.path]} isDisabled={disabledPaths.has(activeSkill.path)} onToggleDisabled={async () => { if (disabledPaths.has(activeSkill.path)) { await handleEnableSkill(activeSkill.path); } else { await handleDisableSkill(activeSkill.path, activeSkill.tool); } await loadDisabled(); }} onSaved={(u) => setSkills((p) => p.map((s) => s.id === u.id ? u : s))} />
            : !loading
            ? (
              <div style={{ flex: 1, display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", gap: 16, color: "var(--text-3)" }}>
                <div style={{ fontSize: 48, opacity: 0.3 }}>{"\u2b21"}</div>
                <div style={{ fontSize: 18, color: "var(--text-2)", fontWeight: 600 }}>Welcome to Agent Desk</div>
                <div style={{ fontSize: 13, color: "var(--text-3)", textAlign: "center", maxWidth: 320, lineHeight: 1.6 }}>
                  Manage your AI agent skills, rules, and configurations in one place.
                </div>

                <div style={{ display: "flex", flexDirection: "column", gap: 8, marginTop: 8, width: 240 }}>
                  <button onClick={handleAddDir} style={onboardingBtnStyle}>
                    {"\u2295"} Add a project directory
                  </button>
                  <button onClick={() => { setShowMarketplace(true); setShowSettings(false); setShowMCP(false); setShowPlugins(false); }} style={onboardingBtnStyle}>
                    {"\u25c7"} Browse marketplace
                  </button>
                  <button onClick={() => { setShowSettings(true); setShowMarketplace(false); setShowMCP(false); setShowPlugins(false); }} style={onboardingBtnStyle}>
                    {"\u2699"} Open settings
                  </button>
                </div>

                <div style={{ fontSize: 12, color: "var(--text-3)", textAlign: "center", maxWidth: 320, lineHeight: 1.7, marginTop: 12 }}>
                  Agent Desk automatically scans:<br />
                  <span style={{ fontFamily: "var(--font-mono)", fontSize: 11, color: "var(--text-2)" }}>
                    ~/.claude/ &nbsp; ~/.cursor/ &nbsp; ~/.gemini/<br />
                    ~/.kiro/ &nbsp; ~/.amp/ &nbsp; ~/.codex/
                  </span>
                </div>

                <div style={{ display: "flex", gap: 24, marginTop: 12, fontSize: 11, color: "var(--text-3)" }}>
                  <span><kbd style={kbdStyle}>{"\u2318"}N</kbd> New skill</span>
                  <span><kbd style={kbdStyle}>{"\u2318"}R</kbd> Refresh</span>
                </div>
              </div>
            )
            : null}
        </div>
      </div>

      {showNew && <NewSkillModal tools={tools} templates={templates} onClose={() => setShowNew(false)} onCreated={(s) => { setSkills((p) => [s, ...p]); setActiveId(s.id); setShowNew(false); }} />}

      <Toast toast={toast} onDismiss={dismissToast} />

      {/* Sync toast for incoming file syncs */}
      <SyncToastNotification toast={syncToast} onDismiss={() => setSyncToast(null)} />

      {/* Sync conflict resolution dialog */}
      {showConflictDialog && syncConflicts.length > 0 && (
        <SyncConflictDialog
          conflicts={syncConflicts}
          onResolve={handleResolveConflict}
          onClose={() => setShowConflictDialog(false)}
        />
      )}
    </>
  );
}

// --- Sync toast component (auto-dismiss, no undo) ---

function SyncToastNotification({ toast, onDismiss }: { toast: SyncToastState | null; onDismiss: () => void }) {
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    if (toast) {
      requestAnimationFrame(() => setVisible(true));
      const timer = setTimeout(() => {
        setVisible(false);
        setTimeout(onDismiss, 200);
      }, 3000);
      return () => clearTimeout(timer);
    } else {
      setVisible(false);
    }
  }, [toast, onDismiss]);

  if (!toast) return null;

  const style: CSSProperties = {
    position: "fixed",
    bottom: 60,
    left: "50%",
    transform: "translateX(-50%)",
    background: "var(--bg-3)",
    border: "1px solid var(--border)",
    borderRadius: "var(--radius)",
    padding: "8px 14px",
    display: "flex",
    alignItems: "center",
    gap: 8,
    boxShadow: "0 4px 16px rgba(0, 0, 0, 0.3)",
    zIndex: 999,
    opacity: visible ? 1 : 0,
    transition: "opacity 0.2s ease",
    pointerEvents: visible ? "auto" : "none",
    fontSize: 12,
    color: "var(--text-2)",
    fontFamily: "var(--font-ui)",
    whiteSpace: "nowrap",
  };

  return <div style={style}>{toast.message}</div>;
}

const kbdStyle: CSSProperties = {
  fontFamily: "var(--font-mono)",
  background: "var(--bg-3)",
  padding: "1px 5px",
  borderRadius: 3,
  border: "1px solid var(--border)",
  fontSize: 10,
};

const onboardingBtnStyle: CSSProperties = {
  background: "transparent",
  border: "1px solid var(--border)",
  borderRadius: "var(--radius)",
  color: "var(--text-2)",
  padding: "8px 14px",
  cursor: "pointer",
  fontSize: 12,
  fontFamily: "var(--font-ui)",
  textAlign: "left",
  transition: "background 0.1s, border-color 0.1s",
};
