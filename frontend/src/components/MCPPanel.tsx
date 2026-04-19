import { useState, useMemo, CSSProperties } from "react";
import { MCPToolConfig, MCPServer } from "../types";

interface MCPPanelProps {
  configs: MCPToolConfig[];
  onClose: () => void;
  onRefresh: () => void;
}

// Color mapping for MCP tools (mirrors KnownTools where applicable)
const toolColors: Record<string, string> = {
  "claude-desktop": "#d4772c",
  "claude-code": "#d4772c",
  "cursor": "#5c8af0",
  "windsurf": "#3ecf8e",
  "copilot": "#f0c040",
  "copilot-intellij": "#f0c040",
  "kiro": "#ff9900",
  "gemini-cli": "#4285f4",
  "amp": "#f97316",
};

export function MCPPanel({ configs, onClose, onRefresh }: MCPPanelProps) {
  const [search, setSearch] = useState("");
  const [collapsedTools, setCollapsedTools] = useState<Record<string, boolean>>({});

  const toggleTool = (tool: string) => {
    setCollapsedTools((prev) => ({ ...prev, [tool]: !prev[tool] }));
  };

  // Filter configs by search across server names
  const filteredConfigs = useMemo(() => {
    if (!search.trim()) return configs;
    const q = search.toLowerCase();
    return configs.map((cfg) => ({
      ...cfg,
      servers: cfg.servers.filter(
        (s) =>
          s.name.toLowerCase().includes(q) ||
          s.command.toLowerCase().includes(q) ||
          (s.url && s.url.toLowerCase().includes(q))
      ),
    }));
  }, [configs, search]);

  const totalServers = configs.reduce((n, c) => n + c.servers.length, 0);

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%", overflow: "hidden" }}>
      {/* Header */}
      <div style={headerStyle}>
        <div style={{ display: "flex", alignItems: "center", gap: 8, flex: 1 }}>
          <span style={{ fontSize: 14, color: "var(--accent)" }}>{"\u2B21"}</span>
          <span style={{ fontSize: 15, fontWeight: 600, color: "var(--text)" }}>
            MCP Servers
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
            {totalServers} server{totalServers !== 1 ? "s" : ""}
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
          placeholder="Filter servers..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          style={searchStyle}
        />
      </div>

      {/* Scrollable content */}
      <div style={{ flex: 1, overflow: "auto", padding: "0 20px 20px" }}>
        {filteredConfigs.map((cfg) => {
          const isCollapsed = collapsedTools[cfg.tool] ?? false;
          const hasServers = cfg.servers.length > 0;
          const color = toolColors[cfg.tool] || "#6b7280";

          return (
            <div key={cfg.tool} style={{ marginTop: 16 }}>
              {/* Tool header */}
              <div
                onClick={() => hasServers && toggleTool(cfg.tool)}
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: 8,
                  cursor: hasServers ? "pointer" : "default",
                  userSelect: "none",
                  padding: "6px 0",
                }}
              >
                <span
                  style={{
                    width: 8,
                    height: 8,
                    borderRadius: "50%",
                    background: hasServers && cfg.exists ? color : "var(--text-3)",
                    opacity: hasServers && cfg.exists ? 1 : 0.4,
                    flexShrink: 0,
                  }}
                />
                <span
                  style={{
                    fontSize: 11,
                    fontWeight: 600,
                    color: hasServers && cfg.exists ? "var(--text)" : "var(--text-3)",
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
                  {!cfg.exists
                    ? "not installed"
                    : hasServers
                    ? `${cfg.servers.length} server${cfg.servers.length !== 1 ? "s" : ""}`
                    : "not configured"}
                </span>
                {hasServers && (
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

              {/* Config file path */}
              <div
                style={{
                  fontSize: 10,
                  fontFamily: "var(--font-mono)",
                  color: "var(--text-3)",
                  opacity: 0.7,
                  padding: "0 0 4px 16px",
                  overflow: "hidden",
                  textOverflow: "ellipsis",
                  whiteSpace: "nowrap",
                }}
                title={cfg.path}
              >
                {cfg.path}
              </div>

              {/* Server entries */}
              {hasServers && !isCollapsed && (
                <div style={{ padding: "4px 0 0 16px" }}>
                  {cfg.servers.map((server) => (
                    <ServerEntry key={server.name} server={server} color={color} />
                  ))}
                </div>
              )}

              {/* Divider */}
              <div style={{ height: 1, background: "var(--border)", marginTop: 8 }} />
            </div>
          );
        })}

        {filteredConfigs.every((c) => c.servers.length === 0) && search.trim() && (
          <div
            style={{
              padding: 40,
              textAlign: "center",
              color: "var(--text-3)",
              fontSize: 12,
            }}
          >
            No servers match "{search}"
          </div>
        )}
      </div>
    </div>
  );
}

// --- Server entry component ---

function ServerEntry({ server, color }: { server: MCPServer; color: string }) {
  const envKeys = server.env ? Object.keys(server.env) : [];
  const isHTTP = !!server.url;

  return (
    <div style={serverCardStyle}>
      {/* Server name */}
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
          }}
        >
          {server.name}
        </span>
      </div>

      {/* Command + args (stdio) */}
      {server.command && (
        <div style={detailRowStyle}>
          <span style={detailLabelStyle}>command</span>
          <span style={detailValueStyle}>{server.command}</span>
        </div>
      )}
      {server.args && server.args.length > 0 && (
        <div style={detailRowStyle}>
          <span style={detailLabelStyle}>args</span>
          <span style={detailValueStyle}>{server.args.join(" ")}</span>
        </div>
      )}

      {/* URL (HTTP transport) */}
      {isHTTP && (
        <div style={detailRowStyle}>
          <span style={detailLabelStyle}>url</span>
          <span style={detailValueStyle}>{server.url}</span>
        </div>
      )}

      {/* Env keys (values masked) */}
      {envKeys.length > 0 && (
        <div style={detailRowStyle}>
          <span style={detailLabelStyle}>env</span>
          <span style={detailValueStyle}>
            {envKeys.map((key) => `${key}=${server.env![key]}`).join(", ")}
          </span>
        </div>
      )}
    </div>
  );
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

const serverCardStyle: CSSProperties = {
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
