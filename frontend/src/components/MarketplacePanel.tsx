import { useState, useEffect, useMemo, useCallback, CSSProperties } from "react";
import {
  GetMarketplaceSources,
  BrowseMarketplace,
  PreviewMarketplaceSkill,
  InstallMarketplaceSkill,
} from "../../wailsjs/go/main/App";
import { MarketplaceSource, MarketplaceSkill, ToolMeta, getToolMeta, formatBytes } from "../types";

interface MarketplacePanelProps {
  tools: ToolMeta[];
  onClose: () => void;
  onInstalled: () => void;
}

export function MarketplacePanel({ tools, onClose, onInstalled }: MarketplacePanelProps) {
  const [sources, setSources] = useState<MarketplaceSource[]>([]);
  const [activeSourceId, setActiveSourceId] = useState<string>("");
  const [skills, setSkills] = useState<MarketplaceSkill[]>([]);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Preview state
  const [selectedSkill, setSelectedSkill] = useState<MarketplaceSkill | null>(null);
  const [previewContent, setPreviewContent] = useState<string>("");
  const [previewLoading, setPreviewLoading] = useState(false);

  // Install state
  const [installedPaths, setInstalledPaths] = useState<Set<string>>(new Set());
  const [installingPath, setInstallingPath] = useState<string | null>(null);
  const [installError, setInstallError] = useState<string | null>(null);

  // Load sources on mount
  useEffect(() => {
    GetMarketplaceSources().then((s) => {
      const srcs = (s ?? []) as unknown as MarketplaceSource[];
      setSources(srcs);
      if (srcs.length > 0) {
        setActiveSourceId(srcs[0].id);
      }
    });
  }, []);

  // Fetch skills when source changes
  useEffect(() => {
    if (!activeSourceId) return;
    setLoading(true);
    setError(null);
    setSkills([]);
    setSelectedSkill(null);
    setPreviewContent("");

    BrowseMarketplace(activeSourceId)
      .then((result) => {
        setSkills((result ?? []) as unknown as MarketplaceSkill[]);
      })
      .catch((err) => {
        setError(err?.message || "Failed to fetch skills");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [activeSourceId]);

  // Filter by search
  const filtered = useMemo(() => {
    if (!search.trim()) return skills;
    const q = search.toLowerCase();
    return skills.filter(
      (s) => s.name.toLowerCase().includes(q) || s.path.toLowerCase().includes(q)
    );
  }, [skills, search]);

  // Preview a skill
  const handleSelectSkill = useCallback(
    (skill: MarketplaceSkill) => {
      setSelectedSkill(skill);
      setPreviewLoading(true);
      setPreviewContent("");

      PreviewMarketplaceSkill(skill.downloadUrl)
        .then((content) => {
          setPreviewContent(content);
        })
        .catch(() => {
          setPreviewContent("-- Failed to load preview --");
        })
        .finally(() => {
          setPreviewLoading(false);
        });
    },
    []
  );

  // Install a skill
  const handleInstall = useCallback(
    async (skill: MarketplaceSkill) => {
      setInstallingPath(skill.path);
      setInstallError(null);

      try {
        // Derive a reasonable filename from the skill path
        const filename = deriveFilename(skill);

        await InstallMarketplaceSkill(skill.downloadUrl, skill.tool, filename);
        setInstalledPaths((prev) => new Set(prev).add(skill.path));
        // Notify parent to refresh skills list
        onInstalled();
      } catch (err: any) {
        setInstallError(err?.message || "Install failed");
      } finally {
        setInstallingPath(null);
      }
    },
    [onInstalled]
  );

  const activeSource = sources.find((s) => s.id === activeSourceId);
  const activeToolMeta = activeSource ? getToolMeta(activeSource.tool, tools) : null;

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%", overflow: "hidden" }}>
      {/* Header */}
      <div style={headerStyle}>
        <div style={{ display: "flex", alignItems: "center", gap: 8, flex: 1 }}>
          <span style={{ fontSize: 14, color: "var(--accent)" }}>{"\u25C7"}</span>
          <span style={{ fontSize: 15, fontWeight: 600, color: "var(--text)" }}>
            Marketplace
          </span>
          {activeToolMeta && (
            <span
              style={{
                display: "inline-flex",
                alignItems: "center",
                gap: 4,
                background: activeToolMeta.color + "20",
                color: activeToolMeta.color,
                padding: "1px 8px",
                borderRadius: "var(--radius)",
                fontSize: 11,
                fontWeight: 500,
                fontFamily: "var(--font-mono)",
              }}
            >
              <span>{activeToolMeta.icon}</span>
              {activeToolMeta.label}
            </span>
          )}
        </div>
        <button onClick={onClose} style={closeButtonStyle}>
          {"\u2715"}
        </button>
      </div>

      {/* Controls: Source selector + Search */}
      <div style={controlsStyle}>
        <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
          <label style={{ fontSize: 11, color: "var(--text-3)", fontWeight: 500, flexShrink: 0 }}>
            Source:
          </label>
          <select
            value={activeSourceId}
            onChange={(e) => setActiveSourceId(e.target.value)}
            style={selectStyle}
          >
            {sources.map((src) => (
              <option key={src.id} value={src.id}>
                {src.name}
              </option>
            ))}
          </select>
        </div>
        <input
          type="text"
          placeholder="Search skills..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          style={searchStyle}
        />
      </div>

      {/* Install error banner */}
      {installError && (
        <div style={errorBannerStyle}>
          {installError}
          <button
            onClick={() => setInstallError(null)}
            style={{
              background: "transparent",
              border: "none",
              color: "var(--red)",
              cursor: "pointer",
              fontSize: 14,
              padding: "0 4px",
              lineHeight: 1,
            }}
          >
            {"\u2715"}
          </button>
        </div>
      )}

      {/* Content area */}
      <div style={{ flex: 1, display: "flex", flexDirection: "column", overflow: "hidden" }}>
        {/* Loading state */}
        {loading && (
          <div style={centerMessageStyle}>
            <span style={{ color: "var(--text-3)" }}>Loading skills...</span>
          </div>
        )}

        {/* Error state */}
        {error && !loading && (
          <div style={centerMessageStyle}>
            <span style={{ color: "var(--red)", fontSize: 12 }}>{error}</span>
            <button
              onClick={() => {
                if (activeSourceId) {
                  setLoading(true);
                  setError(null);
                  BrowseMarketplace(activeSourceId)
                    .then((result) => setSkills((result ?? []) as unknown as MarketplaceSkill[]))
                    .catch((err) => setError(err?.message || "Failed to fetch"))
                    .finally(() => setLoading(false));
                }
              }}
              style={retryButtonStyle}
            >
              Retry
            </button>
          </div>
        )}

        {/* Skill list */}
        {!loading && !error && (
          <>
            <div style={{ flex: 1, overflow: "auto", minHeight: 0 }}>
              {filtered.length === 0 ? (
                <div style={centerMessageStyle}>
                  <span style={{ color: "var(--text-3)", fontSize: 12 }}>
                    {search ? "No matching skills" : "No skills found in this source"}
                  </span>
                </div>
              ) : (
                <div style={{ padding: "4px 16px" }}>
                  <div style={{ fontSize: 11, color: "var(--text-3)", padding: "4px 0 8px", fontFamily: "var(--font-mono)" }}>
                    {filtered.length} skill{filtered.length !== 1 ? "s" : ""} available
                  </div>
                  {filtered.map((skill) => {
                    const isInstalled = installedPaths.has(skill.path);
                    const isInstalling = installingPath === skill.path;
                    const isSelected = selectedSkill?.path === skill.path;
                    const ext = getExtension(skill.path);

                    return (
                      <div
                        key={skill.path}
                        onClick={() => handleSelectSkill(skill)}
                        style={{
                          display: "flex",
                          alignItems: "center",
                          gap: 10,
                          padding: "8px 12px",
                          background: isSelected ? "var(--accent-dim)" : "var(--bg-2)",
                          border: "1px solid " + (isSelected ? "var(--accent)" : "var(--border)"),
                          borderRadius: "var(--radius)",
                          marginBottom: 4,
                          cursor: "pointer",
                          transition: "background 0.1s, border-color 0.15s",
                        }}
                        onMouseEnter={(e) => {
                          if (!isSelected) {
                            e.currentTarget.style.background = "var(--bg-hover)";
                            e.currentTarget.style.borderColor = "var(--border-2)";
                          }
                        }}
                        onMouseLeave={(e) => {
                          if (!isSelected) {
                            e.currentTarget.style.background = "var(--bg-2)";
                            e.currentTarget.style.borderColor = "var(--border)";
                          }
                        }}
                      >
                        {/* Skill info */}
                        <div style={{ flex: 1, minWidth: 0 }}>
                          <div
                            style={{
                              fontSize: 13,
                              fontWeight: 500,
                              color: "var(--text)",
                              overflow: "hidden",
                              textOverflow: "ellipsis",
                              whiteSpace: "nowrap",
                            }}
                          >
                            {skill.name}
                          </div>
                          <div
                            style={{
                              fontSize: 11,
                              color: "var(--text-3)",
                              marginTop: 2,
                              fontFamily: "var(--font-mono)",
                              overflow: "hidden",
                              textOverflow: "ellipsis",
                              whiteSpace: "nowrap",
                            }}
                          >
                            {ext && <span style={{ marginRight: 8 }}>{ext}</span>}
                            {skill.size > 0 && <span>{formatBytes(skill.size)}</span>}
                          </div>
                        </div>

                        {/* Install button or checkmark */}
                        {isInstalled ? (
                          <span
                            style={{
                              color: "var(--green)",
                              fontSize: 14,
                              flexShrink: 0,
                              display: "flex",
                              alignItems: "center",
                              gap: 4,
                              fontWeight: 500,
                            }}
                            title="Installed"
                          >
                            {"\u2713"}
                            <span style={{ fontSize: 11 }}>Installed</span>
                          </span>
                        ) : (
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              handleInstall(skill);
                            }}
                            disabled={isInstalling}
                            style={{
                              background: isInstalling ? "var(--bg-3)" : "var(--green)",
                              border: "none",
                              borderRadius: "var(--radius)",
                              color: isInstalling ? "var(--text-3)" : "#fff",
                              padding: "4px 12px",
                              fontSize: 11,
                              fontWeight: 600,
                              cursor: isInstalling ? "default" : "pointer",
                              fontFamily: "var(--font-ui)",
                              flexShrink: 0,
                              opacity: isInstalling ? 0.7 : 1,
                            }}
                          >
                            {isInstalling ? "Installing..." : "Install"}
                          </button>
                        )}
                      </div>
                    );
                  })}
                </div>
              )}
            </div>

            {/* Preview area */}
            {selectedSkill && (
              <div style={previewContainerStyle}>
                <div style={previewHeaderStyle}>
                  <span style={{ fontSize: 11, fontWeight: 600, color: "var(--text-2)", textTransform: "uppercase", letterSpacing: "0.04em" }}>
                    Preview
                  </span>
                  <span style={{ fontSize: 11, color: "var(--text-3)", fontFamily: "var(--font-mono)" }}>
                    {selectedSkill.name}
                  </span>
                </div>
                <div style={previewBodyStyle}>
                  {previewLoading ? (
                    <span style={{ color: "var(--text-3)", fontSize: 12 }}>Loading preview...</span>
                  ) : (
                    <pre style={previewTextStyle}>{previewContent}</pre>
                  )}
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}

// --- Helpers ---

function getExtension(path: string): string {
  const lastDot = path.lastIndexOf(".");
  if (lastDot < 0) return "";
  return path.substring(lastDot);
}

function deriveFilename(skill: MarketplaceSkill): string {
  // Use the last segment of the path
  const parts = skill.path.split("/");
  const last = parts[parts.length - 1];

  // If it's a directory-style path (e.g., rules/next-js/.cursorrules),
  // the filename might be generic. Use the skill name + appropriate extension instead.
  if (last.startsWith(".") && parts.length > 1) {
    const ext = getExtension(last) || getExtension(skill.path) || ".md";
    return skill.name.replace(/[^a-zA-Z0-9._-]/g, "-").toLowerCase() + ext;
  }

  return last;
}

// --- Styles ---

const headerStyle: CSSProperties = {
  display: "flex",
  alignItems: "center",
  padding: "12px 20px",
  borderBottom: "1px solid var(--border)",
  background: "var(--bg-2)",
  flexShrink: 0,
};

const closeButtonStyle: CSSProperties = {
  background: "transparent",
  border: "1px solid var(--border)",
  borderRadius: "var(--radius)",
  color: "var(--text-2)",
  padding: "3px 10px",
  fontSize: 12,
  cursor: "pointer",
  fontFamily: "var(--font-ui)",
};

const controlsStyle: CSSProperties = {
  display: "flex",
  gap: 12,
  padding: "10px 20px",
  borderBottom: "1px solid var(--border)",
  background: "var(--bg-2)",
  flexShrink: 0,
  alignItems: "center",
};

const selectStyle: CSSProperties = {
  background: "var(--bg-3)",
  border: "1px solid var(--border)",
  borderRadius: "var(--radius)",
  color: "var(--text)",
  padding: "5px 10px",
  fontSize: 12,
  fontFamily: "var(--font-ui)",
  outline: "none",
  cursor: "pointer",
  minWidth: 180,
};

const searchStyle: CSSProperties = {
  flex: 1,
  background: "var(--bg-3)",
  border: "1px solid var(--border)",
  borderRadius: "var(--radius)",
  color: "var(--text)",
  padding: "5px 10px",
  fontSize: 12,
  fontFamily: "var(--font-ui)",
  outline: "none",
};

const centerMessageStyle: CSSProperties = {
  flex: 1,
  display: "flex",
  flexDirection: "column",
  alignItems: "center",
  justifyContent: "center",
  gap: 12,
  padding: 40,
};

const errorBannerStyle: CSSProperties = {
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  padding: "6px 20px",
  background: "rgba(248, 113, 113, 0.1)",
  borderBottom: "1px solid rgba(248, 113, 113, 0.2)",
  color: "var(--red)",
  fontSize: 12,
  flexShrink: 0,
};

const retryButtonStyle: CSSProperties = {
  background: "transparent",
  border: "1px solid var(--border)",
  borderRadius: "var(--radius)",
  color: "var(--text-2)",
  padding: "4px 12px",
  fontSize: 11,
  cursor: "pointer",
  fontFamily: "var(--font-ui)",
};

const previewContainerStyle: CSSProperties = {
  borderTop: "1px solid var(--border)",
  flexShrink: 0,
  maxHeight: 300,
  display: "flex",
  flexDirection: "column",
  overflow: "hidden",
};

const previewHeaderStyle: CSSProperties = {
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  padding: "6px 20px",
  background: "var(--bg-2)",
  borderBottom: "1px solid var(--border)",
  flexShrink: 0,
};

const previewBodyStyle: CSSProperties = {
  flex: 1,
  overflow: "auto",
  padding: "12px 20px",
  background: "var(--bg-3)",
};

const previewTextStyle: CSSProperties = {
  fontFamily: "var(--font-mono)",
  fontSize: 12,
  color: "var(--text-2)",
  lineHeight: 1.6,
  whiteSpace: "pre-wrap",
  wordBreak: "break-word",
  margin: 0,
};
