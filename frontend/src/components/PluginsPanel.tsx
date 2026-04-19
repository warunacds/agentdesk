import { useState, useMemo, CSSProperties } from "react";
import { PluginToolConfig, PluginInfo } from "../types";

interface PluginsPanelProps {
  configs: PluginToolConfig[];
  onClose: () => void;
  onRefresh: () => void;
}

// Color mapping for plugin tools (mirrors KnownTools where applicable)
const toolColors: Record<string, string> = {
  "claude-code": "#d4772c",
  "codex": "#38bdf8",
  "gemini-cli": "#4285f4",
  "opencode": "#6b7280",
};

export function PluginsPanel({ configs, onClose, onRefresh }: PluginsPanelProps) {
  const [search, setSearch] = useState("");
  const [collapsedTools, setCollapsedTools] = useState<Record<string, boolean>>({});

  const toggleTool = (tool: string) => {
    setCollapsedTools((prev) => ({ ...prev, [tool]: !prev[tool] }));
  };

  // Filter configs by search across plugin names
  const filteredConfigs = useMemo(() => {
    if (!search.trim()) return configs;
    const q = search.toLowerCase();
    return configs.map((cfg) => ({
      ...cfg,
      plugins: cfg.plugins.filter(
        (p) =>
          p.name.toLowerCase().includes(q) ||
          p.marketplace.toLowerCase().includes(q)
      ),
    }));
  }, [configs, search]);

  const totalPlugins = configs.reduce((n, c) => n + c.plugins.length, 0);

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%", overflow: "hidden" }}>
      {/* Header */}
      <div style={headerStyle}>
        <div style={{ display: "flex", alignItems: "center", gap: 8, flex: 1 }}>
          <span style={{ fontSize: 14, color: "var(--accent)" }}>{"\u25C8"}</span>
          <span style={{ fontSize: 15, fontWeight: 600, color: "var(--text)" }}>
            Plugins
          </span>
          <span
            style={{
              background: "var(--bg-3)",
              borderRadius: 10,
              padding: "1px 8px",
              fontSize: 11,
              color: "var(--text-3)",
              fontFamily: "var(--font-mono)",
            }}
          >
            {totalPlugins} plugin{totalPlugins !== 1 ? "s" : ""}
          </span>
        </div>
        <button onClick={onRefresh} style={refreshButtonStyle} title="Refresh">
          {"\u21BA"}
        </button>
        <button onClick={onClose} style={closeButtonStyle}>
          {"\u2715"}
        </button>
      </div>

      {/* Search bar */}
      <div style={controlsStyle}>
        <input
          type="text"
          placeholder="Filter plugins..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          style={searchStyle}
        />
      </div>

      {/* Scrollable content */}
      <div style={{ flex: 1, overflow: "auto", padding: "0 20px 20px" }}>
        {filteredConfigs.map((cfg) => {
          const isCollapsed = collapsedTools[cfg.tool] ?? false;
          const hasPlugins = cfg.plugins.length > 0;
          const color = toolColors[cfg.tool] || "#6b7280";

          return (
            <div key={cfg.tool} style={{ marginTop: 16 }}>
              {/* Tool header */}
              <div
                onClick={() => hasPlugins && toggleTool(cfg.tool)}
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: 8,
                  cursor: hasPlugins ? "pointer" : "default",
                  userSelect: "none",
                  padding: "6px 0",
                }}
              >
                <span
                  style={{
                    width: 8,
                    height: 8,
                    borderRadius: "50%",
                    background: hasPlugins && cfg.available ? color : "var(--text-3)",
                    opacity: hasPlugins && cfg.available ? 1 : 0.4,
                    flexShrink: 0,
                  }}
                />
                <span
                  style={{
                    fontSize: 11,
                    fontWeight: 600,
                    color: hasPlugins && cfg.available ? "var(--text)" : "var(--text-3)",
                    textTransform: "uppercase",
                    letterSpacing: "0.04em",
                    flex: 1,
                  }}
                >
                  {cfg.label}
                </span>
                {/* Status badge */}
                <span
                  style={{
                    fontSize: 11,
                    color: "var(--text-3)",
                    fontFamily: "var(--font-mono)",
                  }}
                >
                  {!cfg.available
                    ? "no plugin system"
                    : hasPlugins
                    ? `${cfg.plugins.length} plugin${cfg.plugins.length !== 1 ? "s" : ""}`
                    : "no plugins installed"}
                </span>
                {hasPlugins && (
                  <span
                    style={{
                      fontSize: 9,
                      color: "var(--text-3)",
                      transition: "transform 0.15s",
                      transform: isCollapsed ? "rotate(-90deg)" : "rotate(0deg)",
                    }}
                  >
                    {"\u25BE"}
                  </span>
                )}
              </div>

              {/* Plugin entries */}
              {hasPlugins && !isCollapsed && (
                <div style={{ padding: "4px 0 0 16px" }}>
                  {cfg.plugins.map((plugin, idx) => (
                    <PluginEntry key={`${plugin.name}-${idx}`} plugin={plugin} color={color} />
                  ))}
                </div>
              )}

              {/* Divider */}
              <div style={{ height: 1, background: "var(--border)", marginTop: 8 }} />
            </div>
          );
        })}

        {filteredConfigs.every((c) => c.plugins.length === 0) && search.trim() && (
          <div
            style={{
              padding: 40,
              textAlign: "center",
              color: "var(--text-3)",
              fontSize: 12,
            }}
          >
            No plugins match "{search}"
          </div>
        )}
      </div>
    </div>
  );
}

// --- Plugin entry component ---

function PluginEntry({ plugin, color }: { plugin: PluginInfo; color: string }) {
  const formattedDate = formatPluginDate(plugin.lastUpdated || plugin.installedAt);

  return (
    <div style={pluginCardStyle}>
      {/* Plugin name + version */}
      <div style={{ display: "flex", alignItems: "center", gap: 6, marginBottom: 4 }}>
        <span
          style={{
            width: 4,
            height: 4,
            borderRadius: "50%",
            background: color,
            flexShrink: 0,
          }}
        />
        <span
          style={{
            fontSize: 12,
            fontWeight: 600,
            color: "var(--text)",
            fontFamily: "var(--font-mono)",
            flex: 1,
          }}
        >
          {plugin.name}
        </span>
        {plugin.version && (
          <span
            style={{
              fontSize: 11,
              color: "var(--text-2)",
              fontFamily: "var(--font-mono)",
            }}
          >
            v{plugin.version}
          </span>
        )}
      </div>

      {/* Marketplace + scope */}
      {(plugin.marketplace || plugin.scope) && (
        <div style={detailRowStyle}>
          <span style={detailValueStyle}>
            {[plugin.marketplace, plugin.scope].filter(Boolean).join(" \u00B7 ")}
          </span>
        </div>
      )}

      {/* Updated date */}
      {formattedDate && (
        <div style={detailRowStyle}>
          <span style={detailLabelStyle}>updated</span>
          <span style={detailValueStyle}>{formattedDate}</span>
        </div>
      )}
    </div>
  );
}

// --- Helpers ---

function formatPluginDate(isoDate: string): string {
  if (!isoDate) return "";
  try {
    const d = new Date(isoDate);
    if (isNaN(d.getTime())) return "";
    return d.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" });
  } catch {
    return "";
  }
}

// --- Styles ---

const headerStyle: CSSProperties = {
  display: "flex",
  alignItems: "center",
  padding: "12px 20px",
  borderBottom: "1px solid var(--border)",
  background: "var(--bg-2)",
  flexShrink: 0,
  gap: 8,
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

const refreshButtonStyle: CSSProperties = {
  background: "transparent",
  border: "1px solid var(--border)",
  borderRadius: "var(--radius)",
  color: "var(--text-2)",
  padding: "3px 10px",
  fontSize: 14,
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

const pluginCardStyle: CSSProperties = {
  background: "var(--bg-3)",
  borderRadius: "var(--radius)",
  padding: "8px 12px",
  marginBottom: 4,
};

const detailRowStyle: CSSProperties = {
  display: "flex",
  alignItems: "baseline",
  gap: 8,
  padding: "1px 0",
};

const detailLabelStyle: CSSProperties = {
  fontSize: 10,
  fontWeight: 500,
  color: "var(--text-3)",
  textTransform: "uppercase",
  letterSpacing: "0.04em",
  fontFamily: "var(--font-ui)",
  width: 56,
  flexShrink: 0,
};

const detailValueStyle: CSSProperties = {
  fontSize: 11,
  color: "var(--text-2)",
  fontFamily: "var(--font-mono)",
  overflow: "hidden",
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
};
