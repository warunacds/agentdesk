import { useState, useMemo, useRef, useEffect, useCallback } from "react";
import { Skill, ToolMeta, ToolType, Collection, Category, SyncStatus, SyncManifest, SyncConflict, SearchResult, ValidationWarning, RecentFile, Project, categoryLabels, categoryIcons, getToolMeta } from "../types";

interface SidebarProps {
  skills: Skill[];
  tools: ToolMeta[];
  collections: Collection[];
  activeId: string | null;
  activeCollection: string | null;
  syncStatus: SyncStatus | null;
  teamManagedPaths: Set<string>;
  syncManifest: SyncManifest | null;
  syncConflicts: SyncConflict[];
  syncingPaths: Set<string>;
  flashPaths: Set<string>;
  recentFiles: RecentFile[];
  validationWarnings: Record<string, ValidationWarning[]>;
  selectedPaths: Set<string>;
  onSelect: (skill: Skill) => void;
  onNew: () => void;
  onAddDir: () => void;
  onOpenSettings: () => void;
  onOpenMarketplace: () => void;
  onOpenMCP: () => void;
  onOpenPlugins: () => void;
  onSelectCollection: (id: string | null) => void;
  onCreateCollection: (name: string) => void;
  onRenameCollection: (id: string, name: string) => void;
  onDeleteCollection: (id: string) => void;
  onAddToCollection: (collectionId: string, skillPath: string) => void;
  onRemoveFromCollection: (collectionId: string, skillPath: string) => void;
  onSyncFile: (path: string) => void;
  onStopSyncFile: (path: string) => void;
  onDuplicate: (path: string) => void;
  onClearRecent: () => void;
  onSearchSkills: (query: string) => void;
  searchResults: SearchResult[];
  onSelectSearchResult: (result: SearchResult) => void;
  onToggleSelect: (path: string, meta: boolean) => void;
  onRangeSelect: (path: string) => void;
  onBulkDelete: () => void;
  onBulkAddToCollection: (collectionId: string) => void;
  onBulkSync: () => void;
  disabledPaths: Set<string>;
  onDisableSkill: (path: string, tool: ToolType) => Promise<void>;
  onEnableSkill: (originalPath: string) => Promise<void>;
  onDisableAllForTool: (tool: ToolType) => Promise<void>;
  onEnableAllForTool: (tool: ToolType) => Promise<void>;
  projects: Project[];
  projectSkillsByPath: Record<string, Skill[]>;
  projectOverridesByPath: Record<string, Set<string>>;
  onAddProject: () => Promise<void>;
  onRemoveProject: (id: string) => Promise<void>;
  onRenameProject: (id: string, name: string) => Promise<void>;
  onLoadProjectSkills: (path: string) => Promise<void>;
  onToggleProjectOverride: (projectPath: string, skillPath: string, disabled: boolean) => Promise<void>;
  loading: boolean;
}

const categoryOrder: Category[] = ["agent", "skill", "command", "rule", "config"];

export function Sidebar({
  skills, tools, collections, activeId, activeCollection, syncStatus, teamManagedPaths,
  syncManifest, syncConflicts, syncingPaths, flashPaths,
  recentFiles, validationWarnings, selectedPaths,
  onSelect, onNew, onAddDir, onOpenSettings, onOpenMarketplace, onOpenMCP, onOpenPlugins, onSelectCollection,
  onCreateCollection, onRenameCollection, onDeleteCollection,
  onAddToCollection, onRemoveFromCollection,
  onSyncFile, onStopSyncFile,
  onDuplicate, onClearRecent, onSearchSkills, searchResults, onSelectSearchResult,
  onToggleSelect, onRangeSelect, onBulkDelete, onBulkAddToCollection, onBulkSync,
  disabledPaths, onDisableSkill, onEnableSkill, onDisableAllForTool, onEnableAllForTool,
  projects, projectSkillsByPath, projectOverridesByPath,
  onAddProject, onRemoveProject, onRenameProject, onLoadProjectSkills, onToggleProjectOverride,
  loading,
}: SidebarProps) {
  const [search, setSearch] = useState("");
  const [collapsedTools, setCollapsedTools] = useState<Record<string, boolean>>({});
  const [collapsedCats, setCollapsedCats] = useState<Record<string, boolean>>({});
  const [collectionsCollapsed, setCollectionsCollapsed] = useState(false);
  const [recentCollapsed, setRecentCollapsed] = useState(false);
  const [creatingCollection, setCreatingCollection] = useState(false);
  const [newCollectionName, setNewCollectionName] = useState("");
  const [renamingId, setRenamingId] = useState<string | null>(null);
  const [renameValue, setRenameValue] = useState("");
  const [collectionMenuId, setCollectionMenuId] = useState<string | null>(null);
  const [addToMenuSkillPath, setAddToMenuSkillPath] = useState<string | null>(null);
  const [addToMenuPos, setAddToMenuPos] = useState<{ x: number; y: number } | null>(null);
  const [collectionMenuPos, setCollectionMenuPos] = useState<{ x: number; y: number } | null>(null);
  const [bulkMenuPos, setBulkMenuPos] = useState<{ x: number; y: number } | null>(null);
  const [toolMenu, setToolMenu] = useState<{ tool: ToolType; x: number; y: number } | null>(null);
  const [sidebarWidth, setSidebarWidth] = useState(240);
  const searchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const resizing = useRef(false);

  const handleResizeStart = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    resizing.current = true;
    const startX = e.clientX;
    const startWidth = sidebarWidth;
    const onMove = (ev: MouseEvent) => {
      if (!resizing.current) return;
      const newWidth = Math.max(180, Math.min(500, startWidth + (ev.clientX - startX)));
      setSidebarWidth(newWidth);
    };
    const onUp = () => {
      resizing.current = false;
      document.removeEventListener("mousemove", onMove);
      document.removeEventListener("mouseup", onUp);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
    document.addEventListener("mousemove", onMove);
    document.addEventListener("mouseup", onUp);
  }, [sidebarWidth]);

  const newCollInputRef = useRef<HTMLInputElement>(null);
  const renameInputRef = useRef<HTMLInputElement>(null);

  // Focus inputs when they appear
  useEffect(() => {
    if (creatingCollection && newCollInputRef.current) newCollInputRef.current.focus();
  }, [creatingCollection]);
  useEffect(() => {
    if (renamingId && renameInputRef.current) renameInputRef.current.focus();
  }, [renamingId]);

  // Close menus on outside click
  useEffect(() => {
    const handler = () => {
      setCollectionMenuId(null);
      setCollectionMenuPos(null);
      setAddToMenuSkillPath(null);
      setAddToMenuPos(null);
      setBulkMenuPos(null);
      setToolMenu(null);
    };
    window.addEventListener("click", handler);
    return () => window.removeEventListener("click", handler);
  }, []);

  // Debounced full-text search
  const handleSearchChange = useCallback((value: string) => {
    setSearch(value);
    if (searchTimerRef.current) clearTimeout(searchTimerRef.current);
    if (value.trim()) {
      searchTimerRef.current = setTimeout(() => {
        onSearchSkills(value.trim());
      }, 300);
    }
  }, [onSearchSkills]);

  // Determine displayed skills: if a collection is active, filter to its paths
  const displaySkills = useMemo(() => {
    if (!activeCollection) return skills;
    const coll = collections.find((c) => c.id === activeCollection);
    if (!coll) return skills;
    const pathSet = new Set(coll.skillPaths);
    return skills.filter((s) => pathSet.has(s.path));
  }, [skills, collections, activeCollection]);

  const filtered = useMemo(() => {
    if (!search.trim()) return displaySkills;
    const q = search.toLowerCase();
    return displaySkills.filter(
      (s) => s.name.toLowerCase().includes(q) || s.content.toLowerCase().includes(q)
    );
  }, [displaySkills, search]);

  // Group: tool -> category -> skills[]
  const grouped = useMemo(() => {
    const groups: Record<string, Record<string, Skill[]>> = {};
    for (const s of filtered) {
      if (!groups[s.tool]) groups[s.tool] = {};
      const cat = s.category || "config";
      if (!groups[s.tool][cat]) groups[s.tool][cat] = [];
      groups[s.tool][cat].push(s);
    }
    return groups;
  }, [filtered]);

  const toolOrder = tools.filter((t) => grouped[t.id]).sort((a, b) => {
    const ac = Object.values(grouped[a.id] || {}).reduce((n, arr) => n + arr.length, 0);
    const bc = Object.values(grouped[b.id] || {}).reduce((n, arr) => n + arr.length, 0);
    return bc - ac;
  });

  const toggleTool = (id: string) => {
    setCollapsedTools((prev) => ({ ...prev, [id]: !prev[id] }));
  };

  const toggleCat = (key: string) => {
    setCollapsedCats((prev) => ({ ...prev, [key]: !prev[key] }));
  };

  const totalCount = (toolId: string) =>
    Object.values(grouped[toolId] || {}).reduce((n, arr) => n + arr.length, 0);

  // --- Collection creation ---
  const handleCreateSubmit = () => {
    const name = newCollectionName.trim();
    if (name) {
      onCreateCollection(name);
    }
    setNewCollectionName("");
    setCreatingCollection(false);
  };

  // --- Collection context menu ---
  const handleCollectionContextMenu = (e: React.MouseEvent, id: string) => {
    e.preventDefault();
    e.stopPropagation();
    setCollectionMenuId(id);
    setCollectionMenuPos({ x: e.clientX, y: e.clientY });
  };

  // --- Skill "add to collection" menu ---
  const handleSkillAddMenu = (e: React.MouseEvent, skillPath: string) => {
    e.preventDefault();
    e.stopPropagation();
    setAddToMenuSkillPath(skillPath);
    setAddToMenuPos({ x: e.clientX, y: e.clientY });
  };

  // Check if a skill is in the active collection (for showing remove option)
  const isInActiveCollection = (skillPath: string): boolean => {
    if (!activeCollection) return false;
    const coll = collections.find((c) => c.id === activeCollection);
    return coll ? coll.skillPaths.includes(skillPath) : false;
  };

  // Convert an absolute skill path to a tilde-relative path for manifest comparison
  const toRelPath = (absPath: string): string => {
    // Sync manifest uses tilde-relative paths like "~/.claude/agents/foo.md"
    const home = absPath.match(/^(\/Users\/[^/]+|\/home\/[^/]+)/);
    if (home) return absPath.replace(home[1], "~");
    return absPath;
  };

  // Check if a skill is in the sync manifest
  const isInSyncManifest = (skillPath: string): boolean => {
    if (!syncManifest) return false;
    if (syncManifest.mode === "all") return true;
    const rel = toRelPath(skillPath);
    return syncManifest.syncedPaths.includes(rel);
  };

  // Check if a skill has a sync conflict
  const hasConflict = (skillPath: string): boolean => {
    if (!syncConflicts || syncConflicts.length === 0) return false;
    const rel = toRelPath(skillPath);
    return syncConflicts.some((c) => c.path === rel);
  };

  // Check if a skill is currently syncing
  const isSyncing = (skillPath: string): boolean => {
    return syncingPaths.has(skillPath);
  };

  // Check if a skill path should flash
  const isFlashing = (skillPath: string): boolean => {
    return flashPaths.has(skillPath);
  };

  return (
    <div
      style={{
        width: sidebarWidth,
        minWidth: 180,
        maxWidth: 500,
        background: "var(--bg-2)",
        borderRight: "1px solid var(--border)",
        display: "flex",
        flexDirection: "column",
        height: "100%",
        overflow: "hidden",
        position: "relative",
      }}
    >
      {/* Resize handle */}
      <div
        onMouseDown={handleResizeStart}
        style={{
          position: "absolute",
          top: 0,
          right: -3,
          width: 6,
          height: "100%",
          cursor: "col-resize",
          zIndex: 100,
        }}
      />
      {/* Search */}
      <div style={{ padding: "10px 10px 6px" }}>
        <input
          type="text"
          placeholder="Search skills..."
          value={search}
          onChange={(e) => handleSearchChange(e.target.value)}
          style={{
            width: "100%",
            background: "var(--bg-3)",
            border: "1px solid var(--border)",
            borderRadius: "var(--radius)",
            color: "var(--text)",
            padding: "6px 10px",
            fontSize: 12,
            fontFamily: "var(--font-ui)",
            outline: "none",
          }}
        />
      </div>

      {/* Full-text search results */}
      {search.trim() && searchResults.length > 0 && (
        <div style={{ maxHeight: 200, overflowY: "auto", borderBottom: "1px solid var(--border)", padding: "4px 0" }}>
          <div style={{ padding: "4px 12px", fontSize: 10, color: "var(--text-3)", fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.04em" }}>
            {searchResults.length} match{searchResults.length !== 1 ? "es" : ""}
          </div>
          {searchResults.map((r, i) => (
            <div
              key={`${r.path}-${r.line}-${i}`}
              onClick={() => onSelectSearchResult(r)}
              style={{
                padding: "4px 12px",
                cursor: "pointer",
                fontSize: 11,
                color: "var(--text-2)",
                transition: "background 0.1s",
                display: "flex",
                flexDirection: "column",
                gap: 1,
              }}
              onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-hover)"; }}
              onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
            >
              <span style={{ fontWeight: 500, color: "var(--text)", fontSize: 11, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                {r.name}
                <span style={{ color: "var(--text-3)", fontWeight: 400, marginLeft: 6, fontFamily: "var(--font-mono)", fontSize: 10 }}>
                  L{r.line}
                </span>
              </span>
              <span style={{ fontFamily: "var(--font-mono)", fontSize: 10, color: "var(--text-3)", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                {r.content.trim()}
              </span>
            </div>
          ))}
        </div>
      )}

      {/* Multi-select count bar */}
      {selectedPaths.size > 0 && (
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 8,
            padding: "5px 12px",
            background: "var(--accent-dim)",
            borderBottom: "1px solid var(--border)",
            fontSize: 11,
            color: "var(--accent)",
            fontWeight: 600,
          }}
        >
          <span>{selectedPaths.size} selected</span>
          <div style={{ flex: 1 }} />
          <button
            onClick={(e) => {
              e.stopPropagation();
              setBulkMenuPos({ x: e.clientX, y: e.clientY });
            }}
            style={{
              background: "transparent",
              border: "1px solid var(--accent)",
              borderRadius: "var(--radius)",
              color: "var(--accent)",
              padding: "2px 8px",
              fontSize: 10,
              cursor: "pointer",
              fontFamily: "var(--font-ui)",
            }}
          >
            Actions
          </button>
        </div>
      )}

      {/* Scrollable list area */}
      <div style={{ flex: 1, overflow: "auto", padding: "4px 0" }}>

        {/* === Collections Section === */}
        <div style={{ marginBottom: 4 }}>
          {/* Collections header */}
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: 6,
              padding: "7px 12px",
              cursor: "pointer",
              userSelect: "none",
              fontSize: 11,
              fontWeight: 600,
              color: "var(--text-2)",
              textTransform: "uppercase",
              letterSpacing: "0.04em",
            }}
          >
            <span
              onClick={() => setCollectionsCollapsed((v) => !v)}
              style={{ display: "flex", alignItems: "center", gap: 6, flex: 1 }}
            >
              <span style={{ fontSize: 12, opacity: 0.7 }}>{"\u2750"}</span>
              <span style={{ flex: 1 }}>Collections</span>
              <span
                style={{
                  background: "var(--bg-3)",
                  borderRadius: 10,
                  padding: "0 6px",
                  fontSize: 10,
                  color: "var(--text-3)",
                  minWidth: 18,
                  textAlign: "center",
                }}
              >
                {collections.length}
              </span>
              <span
                style={{
                  fontSize: 9,
                  color: "var(--text-3)",
                  transition: "transform 0.15s",
                  transform: collectionsCollapsed ? "rotate(-90deg)" : "rotate(0deg)",
                }}
              >
                {"\u25be"}
              </span>
            </span>
            <button
              onClick={(e) => {
                e.stopPropagation();
                setCreatingCollection(true);
                setCollectionsCollapsed(false);
              }}
              style={{
                background: "transparent",
                border: "none",
                color: "var(--text-3)",
                cursor: "pointer",
                fontSize: 14,
                padding: "0 2px",
                lineHeight: 1,
              }}
              title="New collection"
            >
              +
            </button>
          </div>

          {!collectionsCollapsed && (
            <>
              {/* "All Skills" item -- clears collection filter */}
              {activeCollection !== null && (
                <div
                  onClick={() => onSelectCollection(null)}
                  style={{
                    padding: "4px 12px 4px 28px",
                    cursor: "pointer",
                    fontSize: 12,
                    color: "var(--text-3)",
                    fontStyle: "italic",
                    transition: "background 0.1s",
                  }}
                  onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-hover)"; }}
                  onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
                >
                  Show all skills
                </div>
              )}

              {/* Collection items */}
              {collections.map((coll) => {
                const isActive = activeCollection === coll.id;
                const isRenaming = renamingId === coll.id;

                if (isRenaming) {
                  return (
                    <div key={coll.id} style={{ padding: "3px 12px 3px 28px" }}>
                      <input
                        ref={renameInputRef}
                        type="text"
                        value={renameValue}
                        onChange={(e) => setRenameValue(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === "Enter") {
                            const name = renameValue.trim();
                            if (name) onRenameCollection(coll.id, name);
                            setRenamingId(null);
                          }
                          if (e.key === "Escape") setRenamingId(null);
                        }}
                        onBlur={() => {
                          const name = renameValue.trim();
                          if (name && name !== coll.name) onRenameCollection(coll.id, name);
                          setRenamingId(null);
                        }}
                        style={{
                          width: "100%",
                          background: "var(--bg-3)",
                          border: "1px solid var(--accent)",
                          borderRadius: "var(--radius)",
                          color: "var(--text)",
                          padding: "3px 6px",
                          fontSize: 12,
                          fontFamily: "var(--font-ui)",
                          outline: "none",
                        }}
                      />
                    </div>
                  );
                }

                return (
                  <div
                    key={coll.id}
                    onClick={() => onSelectCollection(coll.id)}
                    onContextMenu={(e) => handleCollectionContextMenu(e, coll.id)}
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: 6,
                      padding: "4px 12px 4px 28px",
                      cursor: "pointer",
                      fontSize: 12,
                      color: isActive ? "var(--text)" : "var(--text-2)",
                      background: isActive ? "var(--accent-dim)" : "transparent",
                      borderLeft: isActive ? "2px solid var(--accent)" : "2px solid transparent",
                      transition: "background 0.1s",
                    }}
                    onMouseEnter={(e) => {
                      if (!isActive) e.currentTarget.style.background = "var(--bg-hover)";
                    }}
                    onMouseLeave={(e) => {
                      if (!isActive) e.currentTarget.style.background = "transparent";
                    }}
                  >
                    <span style={{ flex: 1, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                      {coll.name}
                    </span>
                    <span
                      style={{
                        background: "var(--bg-3)",
                        borderRadius: 10,
                        padding: "0 5px",
                        fontSize: 10,
                        color: "var(--text-3)",
                        minWidth: 16,
                        textAlign: "center",
                        flexShrink: 0,
                      }}
                    >
                      {coll.skillPaths.length}
                    </span>
                  </div>
                );
              })}

              {/* New collection inline input */}
              {creatingCollection && (
                <div style={{ padding: "3px 12px 3px 28px" }}>
                  <input
                    ref={newCollInputRef}
                    type="text"
                    placeholder="Collection name..."
                    value={newCollectionName}
                    onChange={(e) => setNewCollectionName(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") handleCreateSubmit();
                      if (e.key === "Escape") { setCreatingCollection(false); setNewCollectionName(""); }
                    }}
                    onBlur={() => { handleCreateSubmit(); }}
                    style={{
                      width: "100%",
                      background: "var(--bg-3)",
                      border: "1px solid var(--accent)",
                      borderRadius: "var(--radius)",
                      color: "var(--text)",
                      padding: "3px 6px",
                      fontSize: 12,
                      fontFamily: "var(--font-ui)",
                      outline: "none",
                    }}
                  />
                </div>
              )}

              {collections.length === 0 && !creatingCollection && (
                <div style={{ padding: "6px 12px 6px 28px", fontSize: 11, color: "var(--text-3)" }}>
                  No collections yet
                </div>
              )}
            </>
          )}

          {/* Separator between collections and recent */}
          <div style={{ height: 1, background: "var(--border)", margin: "4px 12px" }} />
        </div>

        {/* === Projects Section === */}
        <div style={{ marginBottom: 4 }}>
          <div
            style={{
              display: "flex",
              alignItems: "center",
              justifyContent: "space-between",
              padding: "7px 12px",
              fontSize: 11,
              fontWeight: 600,
              color: "var(--text-2)",
              textTransform: "uppercase",
              letterSpacing: "0.04em",
            }}
          >
            <span style={{ display: "flex", alignItems: "center", gap: 6, flex: 1 }}>
              <span style={{ fontSize: 12, opacity: 0.7 }}>{"\uD83D\uDCC1"}</span>
              <span style={{ flex: 1 }}>Projects</span>
              <span
                style={{
                  background: "var(--bg-3)",
                  borderRadius: 10,
                  padding: "0 6px",
                  fontSize: 10,
                  color: "var(--text-3)",
                  minWidth: 18,
                  textAlign: "center",
                }}
              >
                {projects.length}
              </span>
            </span>
            <button
              onClick={onAddProject}
              title="Add project"
              style={{
                background: "transparent",
                border: "none",
                color: "var(--text-3)",
                cursor: "pointer",
                fontSize: 14,
                padding: "0 2px",
                lineHeight: 1,
              }}
            >
              +
            </button>
          </div>
          {projects.length === 0 && (
            <div style={{ padding: "6px 12px 6px 28px", fontSize: 11, color: "var(--text-3)" }}>
              No projects pinned
            </div>
          )}
          {projects.map((project) => (
            <ProjectNode
              key={project.id}
              project={project}
              allSkills={skills}
              localSkills={projectSkillsByPath[project.path] ?? []}
              overrides={projectOverridesByPath[project.path] ?? new Set()}
              onExpand={() => onLoadProjectSkills(project.path)}
              onRemove={() => onRemoveProject(project.id)}
              onRename={(name) => onRenameProject(project.id, name)}
              onToggleOverride={(skillPath, disabled) => onToggleProjectOverride(project.path, skillPath, disabled)}
              onSelect={onSelect}
            />
          ))}
          {projects.length > 0 && (
            <div style={{ padding: "6px 12px", fontSize: 10, color: "var(--text-3)", fontStyle: "italic", lineHeight: 1.4 }}>
              View-only organization. Agents still see all global skills at runtime.
            </div>
          )}
          {/* Separator between projects and recent */}
          <div style={{ height: 1, background: "var(--border)", margin: "4px 12px" }} />
        </div>

        {/* === Recent Section === */}
        {recentFiles.length > 0 && (
          <div style={{ marginBottom: 4 }}>
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: 6,
                padding: "7px 12px",
                cursor: "pointer",
                userSelect: "none",
                fontSize: 11,
                fontWeight: 600,
                color: "var(--text-2)",
                textTransform: "uppercase",
                letterSpacing: "0.04em",
              }}
            >
              <span
                onClick={() => setRecentCollapsed((v) => !v)}
                style={{ display: "flex", alignItems: "center", gap: 6, flex: 1 }}
              >
                <span style={{ fontSize: 12, opacity: 0.7 }}>{"\u23F1"}</span>
                <span style={{ flex: 1 }}>Recent</span>
                <span
                  style={{
                    fontSize: 9,
                    color: "var(--text-3)",
                    transition: "transform 0.15s",
                    transform: recentCollapsed ? "rotate(-90deg)" : "rotate(0deg)",
                  }}
                >
                  {"\u25BE"}
                </span>
              </span>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onClearRecent();
                }}
                style={{
                  background: "transparent",
                  border: "none",
                  color: "var(--text-3)",
                  cursor: "pointer",
                  fontSize: 12,
                  padding: "0 2px",
                  lineHeight: 1,
                }}
                title="Clear recent"
              >
                {"\u2715"}
              </button>
            </div>

            {!recentCollapsed &&
              recentFiles.slice(0, 5).map((rf) => {
                const matchingSkill = skills.find((s) => s.path === rf.path);
                return (
                  <div
                    key={rf.path}
                    onClick={() => {
                      if (matchingSkill) onSelect(matchingSkill);
                    }}
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: 6,
                      padding: "4px 12px 4px 28px",
                      cursor: matchingSkill ? "pointer" : "default",
                      fontSize: 12,
                      color: matchingSkill ? "var(--text-2)" : "var(--text-3)",
                      transition: "background 0.1s",
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                      whiteSpace: "nowrap",
                      opacity: matchingSkill ? 1 : 0.5,
                    }}
                    onMouseEnter={(e) => { if (matchingSkill) e.currentTarget.style.background = "var(--bg-hover)"; }}
                    onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
                  >
                    <span style={{ flex: 1, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                      {rf.name}
                    </span>
                    <span style={{ fontSize: 10, color: "var(--text-3)", flexShrink: 0 }}>
                      {rf.tool}
                    </span>
                  </div>
                );
              })}

            {/* Separator between recent and tools */}
            <div style={{ height: 1, background: "var(--border)", margin: "4px 12px" }} />
          </div>
        )}

        {/* === Tool-grouped skill list === */}
        {loading && skills.length === 0 ? (
          <div style={{ padding: "20px 16px", color: "var(--text-3)", fontSize: 12, textAlign: "center" }}>
            Scanning...
          </div>
        ) : filtered.length === 0 ? (
          <div style={{ padding: "20px 16px", color: "var(--text-3)", fontSize: 12, textAlign: "center" }}>
            {search ? "No matches" : activeCollection ? "No skills in this collection" : "No skills found"}
          </div>
        ) : (
          toolOrder.map((toolMeta) => {
            const isToolCollapsed = collapsedTools[toolMeta.id] ?? false;
            const toolCats = grouped[toolMeta.id] ?? {};
            return (
              <div key={toolMeta.id} style={{ marginBottom: 2 }}>
                {/* Tool header */}
                <div
                  onClick={() => toggleTool(toolMeta.id)}
                  onContextMenu={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    setToolMenu({ tool: toolMeta.id, x: e.clientX, y: e.clientY });
                  }}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 6,
                    padding: "7px 12px",
                    cursor: "pointer",
                    userSelect: "none",
                    fontSize: 11,
                    fontWeight: 600,
                    color: "var(--text-2)",
                    textTransform: "uppercase",
                    letterSpacing: "0.04em",
                  }}
                >
                  <span
                    style={{
                      width: 8,
                      height: 8,
                      borderRadius: "50%",
                      background: toolMeta.color,
                      flexShrink: 0,
                    }}
                  />
                  <span style={{ flex: 1 }}>{toolMeta.label}</span>
                  <span
                    style={{
                      background: "var(--bg-3)",
                      borderRadius: 10,
                      padding: "0 6px",
                      fontSize: 10,
                      color: "var(--text-3)",
                      minWidth: 18,
                      textAlign: "center",
                    }}
                  >
                    {totalCount(toolMeta.id)}
                  </span>
                  <span
                    style={{
                      fontSize: 9,
                      color: "var(--text-3)",
                      transition: "transform 0.15s",
                      transform: isToolCollapsed ? "rotate(-90deg)" : "rotate(0deg)",
                    }}
                  >
                    {"\u25be"}
                  </span>
                </div>

                {/* Category groups */}
                {!isToolCollapsed &&
                  categoryOrder
                    .filter((cat) => toolCats[cat] && toolCats[cat].length > 0)
                    .map((cat) => {
                      const catKey = `${toolMeta.id}:${cat}`;
                      const isCatCollapsed = collapsedCats[catKey] ?? false;
                      const items = toolCats[cat];
                      return (
                        <div key={catKey}>
                          {/* Category header */}
                          <div
                            onClick={() => toggleCat(catKey)}
                            style={{
                              display: "flex",
                              alignItems: "center",
                              gap: 5,
                              padding: "4px 12px 4px 22px",
                              cursor: "pointer",
                              userSelect: "none",
                              fontSize: 10,
                              fontWeight: 500,
                              color: "var(--text-3)",
                              letterSpacing: "0.03em",
                            }}
                          >
                            <span style={{ fontSize: 10 }}>{categoryIcons[cat]}</span>
                            <span style={{ flex: 1 }}>{categoryLabels[cat]}</span>
                            <span
                              style={{
                                fontSize: 9,
                                color: "var(--text-3)",
                                opacity: 0.6,
                              }}
                            >
                              {items.length}
                            </span>
                            <span
                              style={{
                                fontSize: 8,
                                color: "var(--text-3)",
                                transition: "transform 0.15s",
                                transform: isCatCollapsed ? "rotate(-90deg)" : "rotate(0deg)",
                              }}
                            >
                              {"\u25be"}
                            </span>
                          </div>

                          {/* Items */}
                          {!isCatCollapsed &&
                            items.map((skill) => {
                              const synced = isInSyncManifest(skill.path);
                              const conflicted = hasConflict(skill.path);
                              const syncing = isSyncing(skill.path);
                              const flashing = isFlashing(skill.path);
                              const isSelected = selectedPaths.has(skill.path);
                              const hasWarnings = validationWarnings[skill.path] && validationWarnings[skill.path].length > 0;
                              return (
                              <div
                                key={skill.id}
                                onClick={(e) => {
                                  if (e.metaKey || e.ctrlKey) {
                                    onToggleSelect(skill.path, true);
                                  } else if (e.shiftKey) {
                                    onRangeSelect(skill.path);
                                  } else {
                                    onSelect(skill);
                                  }
                                }}
                                onContextMenu={(e) => handleSkillAddMenu(e, skill.path)}
                                style={{
                                  display: "flex",
                                  alignItems: "center",
                                  padding: "4px 12px 4px 34px",
                                  cursor: "pointer",
                                  fontSize: 12,
                                  color: skill.id === activeId ? "var(--text)" : "var(--text-2)",
                                  background: flashing
                                    ? "rgba(62, 207, 142, 0.15)"
                                    : isSelected
                                    ? "var(--accent-dim)"
                                    : skill.id === activeId
                                    ? "var(--accent-dim)"
                                    : "transparent",
                                  borderLeft: isSelected
                                    ? "2px solid var(--accent)"
                                    : skill.id === activeId ? "2px solid var(--accent)" : "2px solid transparent",
                                  overflow: "hidden",
                                  textOverflow: "ellipsis",
                                  whiteSpace: "nowrap",
                                  transition: "background 1s ease",
                                  gap: 4,
                                  opacity: disabledPaths.has(skill.path) ? 0.45 : 1,
                                }}
                                onMouseEnter={(e) => {
                                  if (skill.id !== activeId && !flashing && !isSelected) e.currentTarget.style.background = "var(--bg-hover)";
                                }}
                                onMouseLeave={(e) => {
                                  if (skill.id !== activeId && !flashing && !isSelected) e.currentTarget.style.background = "transparent";
                                }}
                              >
                                {/* Team-managed lock icon */}
                                {teamManagedPaths.has(skill.path) && (
                                  <span
                                    title="Managed by team"
                                    style={{
                                      fontSize: 10,
                                      color: "var(--accent)",
                                      flexShrink: 0,
                                      opacity: 0.8,
                                    }}
                                  >
                                    {"\uD83D\uDD12"}
                                  </span>
                                )}
                                {/* Sync indicators */}
                                {syncing && (
                                  <span
                                    title="Syncing..."
                                    style={{
                                      fontSize: 10,
                                      flexShrink: 0,
                                      color: "var(--accent)",
                                      animation: "spin 1s linear infinite",
                                    }}
                                  >
                                    {"\u21BB"}
                                  </span>
                                )}
                                {!syncing && conflicted && (
                                  <span
                                    title="Sync conflict"
                                    style={{
                                      fontSize: 10,
                                      flexShrink: 0,
                                      color: "var(--yellow)",
                                    }}
                                  >
                                    {"\u26A0"}
                                  </span>
                                )}
                                {!syncing && !conflicted && synced && (
                                  <span
                                    title="Synced"
                                    style={{
                                      fontSize: 9,
                                      flexShrink: 0,
                                      color: "var(--text-3)",
                                      opacity: 0.7,
                                    }}
                                  >
                                    {"\u2601"}
                                  </span>
                                )}
                                {skill.directory && (
                                  <span
                                    title="Folder skill"
                                    style={{
                                      fontSize: 10,
                                      flexShrink: 0,
                                      color: "var(--text-3)",
                                    }}
                                  >
                                    {"\uD83D\uDCC1"}
                                  </span>
                                )}
                                <span style={{ flex: 1, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                                  {skill.name}
                                </span>
                                {/* Validation warning dot */}
                                {hasWarnings && (
                                  <span
                                    title={validationWarnings[skill.path].map((w) => w.message).join("\n")}
                                    style={{
                                      width: 6,
                                      height: 6,
                                      borderRadius: "50%",
                                      background: "var(--yellow)",
                                      flexShrink: 0,
                                    }}
                                  />
                                )}
                                {/* Multi-tool dots for symlinked skills */}
                                {skill.tools && skill.tools.length > 1 && (
                                  <span style={{ display: "flex", gap: 2, marginLeft: 4, flexShrink: 0 }}>
                                    {skill.tools.map((t) => {
                                      const dotMeta = getToolMeta(t, tools);
                                      return (
                                        <span
                                          key={t}
                                          title={dotMeta.label}
                                          style={{
                                            width: 6,
                                            height: 6,
                                            borderRadius: "50%",
                                            background: dotMeta.color,
                                          }}
                                        />
                                      );
                                    })}
                                  </span>
                                )}
                                {/* Remove from collection button (only visible when viewing a collection) */}
                                {activeCollection && isInActiveCollection(skill.path) && (
                                  <button
                                    onClick={(e) => {
                                      e.stopPropagation();
                                      onRemoveFromCollection(activeCollection, skill.path);
                                    }}
                                    style={{
                                      background: "transparent",
                                      border: "none",
                                      color: "var(--text-3)",
                                      cursor: "pointer",
                                      fontSize: 11,
                                      padding: "0 2px",
                                      lineHeight: 1,
                                      flexShrink: 0,
                                      opacity: 0.6,
                                    }}
                                    title="Remove from collection"
                                  >
                                    {"\u2212"}
                                  </button>
                                )}
                              </div>
                              );
                            })}
                        </div>
                      );
                    })}
              </div>
            );
          })
        )}
      </div>

      {/* Footer */}
      <div
        style={{
          borderTop: "1px solid var(--border)",
          display: "flex",
          flexDirection: "column",
          gap: 0,
          padding: "8px",
        }}
      >
        {/* New skill button */}
        <button
          onClick={onNew}
          style={{
            width: "100%",
            background: "var(--accent)",
            border: "none",
            borderRadius: "var(--radius)",
            color: "#fff",
            padding: "7px 0",
            fontSize: 12,
            fontWeight: 600,
            cursor: "pointer",
            fontFamily: "var(--font-ui)",
          }}
        >
          + New Skill
        </button>
        {/* Action bar */}
        <div
          style={{
            display: "flex",
            marginTop: 6,
            gap: 4,
          }}
        >
          {/* Sync status dot */}
          {syncStatus?.running && (
            <span
              title={
                syncStatus.connectedPeers > 0
                  ? `Connected to ${syncStatus.connectedPeers} peer${syncStatus.connectedPeers !== 1 ? "s" : ""}`
                  : "No peers connected"
              }
              style={{
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                width: 20,
              }}
            >
              <span style={{
                width: 7,
                height: 7,
                borderRadius: "50%",
                background: syncStatus.connectedPeers > 0 ? "var(--green)" : "var(--text-3)",
              }} />
            </span>
          )}
          <button
            onClick={onOpenMarketplace}
            style={footerIconStyle}
            title="Marketplace"
          >
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"><path d="M6 2L3 6v14a2 2 0 002 2h14a2 2 0 002-2V6l-3-4z"/><path d="M3 6h18"/><path d="M16 10a4 4 0 01-8 0"/></svg>
          </button>
          <button
            onClick={onOpenMCP}
            style={footerIconStyle}
            title="MCP Servers"
          >
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"><rect x="2" y="2" width="20" height="8" rx="2"/><rect x="2" y="14" width="20" height="8" rx="2"/><circle cx="6" cy="6" r="1" fill="currentColor" stroke="none"/><circle cx="6" cy="18" r="1" fill="currentColor" stroke="none"/></svg>
          </button>
          <button
            onClick={onOpenPlugins}
            style={footerIconStyle}
            title="Plugins"
          >
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"><path d="M14.7 6.3a1 1 0 000 1.4l1.6 1.6a1 1 0 001.4 0l3.77-3.77a6 6 0 01-7.94 7.94l-6.91 6.91a2.12 2.12 0 01-3-3l6.91-6.91a6 6 0 017.94-7.94l-3.76 3.76z"/></svg>
          </button>
          <button
            onClick={onOpenSettings}
            style={footerIconStyle}
            title="Settings"
          >
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-4 0v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 010-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 012.83-2.83l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 014 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 010 4h-.09a1.65 1.65 0 00-1.51 1z"/></svg>
          </button>
        </div>
      </div>

      {/* === Context menu: Collection right-click === */}
      {collectionMenuId && collectionMenuPos && (
        <div
          onClick={(e) => e.stopPropagation()}
          style={{
            position: "fixed",
            top: collectionMenuPos.y,
            left: collectionMenuPos.x,
            zIndex: 200,
            background: "var(--bg-3)",
            border: "1px solid var(--border)",
            borderRadius: "var(--radius)",
            padding: "4px 0",
            minWidth: 140,
            boxShadow: "0 4px 12px rgba(0,0,0,0.4)",
          }}
        >
          <div
            onClick={() => {
              const coll = collections.find((c) => c.id === collectionMenuId);
              if (coll) {
                setRenameValue(coll.name);
                setRenamingId(coll.id);
              }
              setCollectionMenuId(null);
              setCollectionMenuPos(null);
            }}
            style={contextMenuItemStyle}
            onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-hover)"; }}
            onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
          >
            Rename
          </div>
          <div
            onClick={() => {
              onDeleteCollection(collectionMenuId);
              setCollectionMenuId(null);
              setCollectionMenuPos(null);
            }}
            style={{ ...contextMenuItemStyle, color: "var(--red)" }}
            onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-hover)"; }}
            onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
          >
            Delete
          </div>
        </div>
      )}

      {/* === Context menu: Skill right-click "Add to collection" + sync + duplicate + disable === */}
      {addToMenuSkillPath && addToMenuPos && (
        <div
          onClick={(e) => e.stopPropagation()}
          style={{
            position: "fixed",
            top: addToMenuPos.y,
            left: addToMenuPos.x,
            zIndex: 200,
            background: "var(--bg-3)",
            border: "1px solid var(--border)",
            borderRadius: "var(--radius)",
            padding: "4px 0",
            minWidth: 160,
            maxHeight: 300,
            overflowY: "auto",
            boxShadow: "0 4px 12px rgba(0,0,0,0.4)",
          }}
        >
          {/* Duplicate option */}
          <div
            onClick={() => {
              onDuplicate(addToMenuSkillPath);
              setAddToMenuSkillPath(null);
              setAddToMenuPos(null);
            }}
            style={contextMenuItemStyle}
            onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-hover)"; }}
            onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
          >
            Duplicate
          </div>
          {/* Enable / Disable option */}
          {(() => {
            const path = addToMenuSkillPath;
            const isDisabled = disabledPaths.has(path);
            const targetSkill = skills.find((s) => s.path === path);
            return (
              <div
                onClick={async () => {
                  setAddToMenuSkillPath(null);
                  setAddToMenuPos(null);
                  if (isDisabled) {
                    await onEnableSkill(path);
                  } else if (targetSkill) {
                    await onDisableSkill(path, targetSkill.tool);
                  }
                }}
                style={contextMenuItemStyle}
                onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-hover)"; }}
                onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
              >
                {isDisabled ? "Enable" : "Disable"}
              </div>
            );
          })()}
          <div style={{ height: 1, background: "var(--border)", margin: "2px 0" }} />
          {/* Sync options (only when sync is running) */}
          {syncStatus?.running && (
            <>
              {isInSyncManifest(addToMenuSkillPath) ? (
                <div
                  onClick={() => {
                    onStopSyncFile(addToMenuSkillPath);
                    setAddToMenuSkillPath(null);
                    setAddToMenuPos(null);
                  }}
                  style={contextMenuItemStyle}
                  onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-hover)"; }}
                  onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
                >
                  Stop syncing this file
                </div>
              ) : (
                <div
                  onClick={() => {
                    onSyncFile(addToMenuSkillPath);
                    setAddToMenuSkillPath(null);
                    setAddToMenuPos(null);
                  }}
                  style={contextMenuItemStyle}
                  onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-hover)"; }}
                  onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
                >
                  Sync this file
                </div>
              )}
              <div style={{ height: 1, background: "var(--border)", margin: "2px 0" }} />
            </>
          )}
          <div style={{ padding: "4px 10px", fontSize: 10, color: "var(--text-3)", fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.04em" }}>
            Add to collection
          </div>
          {collections.length === 0 ? (
            <div style={{ padding: "6px 10px", fontSize: 11, color: "var(--text-3)" }}>
              No collections yet
            </div>
          ) : (
            collections.map((coll) => {
              const alreadyIn = coll.skillPaths.includes(addToMenuSkillPath);
              return (
                <div
                  key={coll.id}
                  onClick={() => {
                    if (!alreadyIn) {
                      onAddToCollection(coll.id, addToMenuSkillPath);
                    }
                    setAddToMenuSkillPath(null);
                    setAddToMenuPos(null);
                  }}
                  style={{
                    ...contextMenuItemStyle,
                    color: alreadyIn ? "var(--text-3)" : "var(--text-2)",
                    cursor: alreadyIn ? "default" : "pointer",
                    display: "flex",
                    alignItems: "center",
                    gap: 6,
                  }}
                  onMouseEnter={(e) => { if (!alreadyIn) e.currentTarget.style.background = "var(--bg-hover)"; }}
                  onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
                >
                  <span style={{ flex: 1, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                    {coll.name}
                  </span>
                  {alreadyIn && (
                    <span style={{ fontSize: 10, color: "var(--text-3)" }}>{"\u2713"}</span>
                  )}
                </div>
              );
            })
          )}
        </div>
      )}

      {/* === Context menu: Bulk operations (multi-select) === */}
      {bulkMenuPos && selectedPaths.size > 0 && (
        <div
          onClick={(e) => e.stopPropagation()}
          style={{
            position: "fixed",
            top: bulkMenuPos.y,
            left: bulkMenuPos.x,
            zIndex: 200,
            background: "var(--bg-3)",
            border: "1px solid var(--border)",
            borderRadius: "var(--radius)",
            padding: "4px 0",
            minWidth: 180,
            maxHeight: 300,
            overflowY: "auto",
            boxShadow: "0 4px 12px rgba(0,0,0,0.4)",
          }}
        >
          <div
            onClick={() => {
              onBulkDelete();
              setBulkMenuPos(null);
            }}
            style={{ ...contextMenuItemStyle, color: "var(--red)" }}
            onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-hover)"; }}
            onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
          >
            Delete {selectedPaths.size} skill{selectedPaths.size !== 1 ? "s" : ""}
          </div>
          {syncStatus?.running && (
            <div
              onClick={() => {
                onBulkSync();
                setBulkMenuPos(null);
              }}
              style={contextMenuItemStyle}
              onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-hover)"; }}
              onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
            >
              Sync {selectedPaths.size} file{selectedPaths.size !== 1 ? "s" : ""}
            </div>
          )}
          {collections.length > 0 && (
            <>
              <div style={{ height: 1, background: "var(--border)", margin: "2px 0" }} />
              <div style={{ padding: "4px 10px", fontSize: 10, color: "var(--text-3)", fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.04em" }}>
                Add {selectedPaths.size} to collection
              </div>
              {collections.map((coll) => (
                <div
                  key={coll.id}
                  onClick={() => {
                    onBulkAddToCollection(coll.id);
                    setBulkMenuPos(null);
                  }}
                  style={contextMenuItemStyle}
                  onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-hover)"; }}
                  onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
                >
                  {coll.name}
                </div>
              ))}
            </>
          )}
        </div>
      )}

      {/* === Context menu: Tool header right-click (disable/enable all for tool) === */}
      {toolMenu && (
        <div
          onClick={(e) => e.stopPropagation()}
          style={{
            position: "fixed",
            top: toolMenu.y,
            left: toolMenu.x,
            zIndex: 200,
            background: "var(--bg-3)",
            border: "1px solid var(--border)",
            borderRadius: "var(--radius)",
            padding: "4px 0",
            minWidth: 180,
            boxShadow: "0 4px 12px rgba(0,0,0,0.4)",
          }}
        >
          <div
            onClick={async () => {
              const t = toolMenu.tool;
              setToolMenu(null);
              await onDisableAllForTool(t);
            }}
            style={contextMenuItemStyle}
            onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-hover)"; }}
            onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
          >
            Disable all for tool
          </div>
          <div
            onClick={async () => {
              const t = toolMenu.tool;
              setToolMenu(null);
              await onEnableAllForTool(t);
            }}
            style={contextMenuItemStyle}
            onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-hover)"; }}
            onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
          >
            Enable all for tool
          </div>
        </div>
      )}
    </div>
  );
}

// === ProjectNode: expandable row rendering Local + Global skills for a pinned project ===

function ProjectNode({
  project, allSkills, localSkills, overrides, onExpand, onRemove, onRename, onToggleOverride, onSelect,
}: {
  project: Project;
  allSkills: Skill[];
  localSkills: Skill[];
  overrides: Set<string>;
  onExpand: () => void;
  onRemove: () => void;
  onRename: (name: string) => void;
  onToggleOverride: (skillPath: string, disabled: boolean) => void;
  onSelect: (skill: Skill) => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const [renaming, setRenaming] = useState(false);
  const [renameValue, setRenameValue] = useState(project.name);
  const renameRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (renaming && renameRef.current) renameRef.current.focus();
  }, [renaming]);

  const prefix = project.path.endsWith("/") ? project.path : project.path + "/";
  const globalSkills = allSkills.filter((s) => !s.path.startsWith(prefix));
  const disabledCount = globalSkills.filter((s) => overrides.has(s.path)).length;

  const toggleExpand = () => {
    const next = !expanded;
    setExpanded(next);
    if (next) onExpand();
  };

  const handleRenameSubmit = () => {
    const name = renameValue.trim();
    if (name && name !== project.name) onRename(name);
    setRenaming(false);
  };

  return (
    <div>
      {renaming ? (
        <div style={{ padding: "3px 12px 3px 28px" }}>
          <input
            ref={renameRef}
            type="text"
            value={renameValue}
            onChange={(e) => setRenameValue(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") handleRenameSubmit();
              if (e.key === "Escape") { setRenaming(false); setRenameValue(project.name); }
            }}
            onBlur={handleRenameSubmit}
            style={{
              width: "100%",
              background: "var(--bg-3)",
              border: "1px solid var(--accent)",
              borderRadius: "var(--radius)",
              color: "var(--text)",
              padding: "3px 6px",
              fontSize: 12,
              fontFamily: "var(--font-ui)",
              outline: "none",
            }}
          />
        </div>
      ) : (
        <div
          onClick={toggleExpand}
          onDoubleClick={(e) => {
            e.stopPropagation();
            setRenameValue(project.name);
            setRenaming(true);
          }}
          style={{
            display: "flex",
            alignItems: "center",
            padding: "4px 12px 4px 28px",
            cursor: "pointer",
            fontSize: 12,
            color: "var(--text-2)",
            gap: 6,
            transition: "background 0.1s",
          }}
          onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-hover)"; }}
          onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
          title={project.path}
        >
          <span style={{ color: "var(--text-3)", fontSize: 9, width: 10, flexShrink: 0 }}>
            {expanded ? "\u25BE" : "\u25B8"}
          </span>
          <span style={{ flex: 1, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
            {project.name}
          </span>
          <button
            onClick={(e) => {
              e.stopPropagation();
              if (confirm(`Remove project "${project.name}"?`)) onRemove();
            }}
            style={{
              background: "transparent",
              border: "none",
              color: "var(--text-3)",
              cursor: "pointer",
              fontSize: 12,
              padding: "0 2px",
              lineHeight: 1,
              flexShrink: 0,
              opacity: 0.6,
            }}
            title="Remove project"
          >
            {"\u00D7"}
          </button>
        </div>
      )}
      {expanded && (
        <div style={{ paddingLeft: 34, paddingRight: 12 }}>
          <div
            style={{
              padding: "4px 0 2px 0",
              fontSize: 10,
              fontWeight: 500,
              color: "var(--text-3)",
              textTransform: "uppercase",
              letterSpacing: "0.04em",
            }}
          >
            Local ({localSkills.length})
          </div>
          {localSkills.length === 0 && (
            <div style={{ padding: "2px 0", fontSize: 11, color: "var(--text-3)" }}>
              No local skills found
            </div>
          )}
          {localSkills.map((s) => (
            <div
              key={s.id}
              onClick={() => onSelect(s)}
              style={{
                display: "flex",
                alignItems: "center",
                gap: 5,
                padding: "3px 0",
                fontSize: 11,
                color: "var(--text-2)",
                cursor: "pointer",
                overflow: "hidden",
                textOverflow: "ellipsis",
                whiteSpace: "nowrap",
              }}
              onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-hover)"; }}
              onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
            >
              <span style={{ fontSize: 10, color: "var(--text-3)", flexShrink: 0 }}>{"\uD83D\uDCC4"}</span>
              <span style={{ flex: 1, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                {s.name}
              </span>
            </div>
          ))}
          <div
            style={{
              padding: "6px 0 2px 0",
              fontSize: 10,
              fontWeight: 500,
              color: "var(--text-3)",
              textTransform: "uppercase",
              letterSpacing: "0.04em",
            }}
          >
            Global ({disabledCount} disabled)
          </div>
          {globalSkills.map((s) => {
            const isDisabled = overrides.has(s.path);
            return (
              <div
                key={s.id}
                onClick={() => onSelect(s)}
                onContextMenu={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  onToggleOverride(s.path, !isDisabled);
                }}
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: 5,
                  padding: "3px 0",
                  fontSize: 11,
                  color: "var(--text-2)",
                  cursor: "pointer",
                  opacity: isDisabled ? 0.45 : 1,
                  overflow: "hidden",
                  textOverflow: "ellipsis",
                  whiteSpace: "nowrap",
                }}
                title={isDisabled ? "Right-click to enable for this project" : "Right-click to disable for this project"}
                onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-hover)"; }}
                onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
              >
                <span style={{ fontSize: 10, color: "var(--text-3)", flexShrink: 0 }}>{"\uD83D\uDCC4"}</span>
                <span style={{ flex: 1, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                  {s.name}
                </span>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

const contextMenuItemStyle: React.CSSProperties = {
  padding: "6px 10px",
  fontSize: 12,
  color: "var(--text-2)",
  cursor: "pointer",
  transition: "background 0.1s",
};

const footerActionStyle: React.CSSProperties = {
  flex: 1,
  background: "var(--bg-3)",
  border: "1px solid var(--border)",
  borderRadius: "var(--radius)",
  color: "var(--text-3)",
  padding: "5px 0",
  fontSize: 11,
  cursor: "pointer",
  fontFamily: "var(--font-ui)",
  transition: "color 0.1s, background 0.1s",
};

const footerIconStyle: React.CSSProperties = {
  flex: 1,
  display: "inline-flex",
  alignItems: "center",
  justifyContent: "center",
  background: "var(--bg-3)",
  border: "1px solid var(--border)",
  borderRadius: "var(--radius)",
  color: "var(--text-3)",
  padding: "6px 0",
  cursor: "pointer",
  transition: "background 0.1s",
};
