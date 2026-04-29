import { useState, useEffect } from "react";
import { GetDefaultScanPaths, SelectDirectoryForTool, GetVersion } from "../../wailsjs/go/main/App";
import { Settings, ToolMeta, ToolType, SyncStatus } from "../types";
import { SyncPanel } from "./SyncPanel";

interface SettingsPanelProps {
  settings: Settings;
  tools: ToolMeta[];
  syncStatus: SyncStatus | null;
  onSave: (settings: Settings) => Promise<void>;
  onClose: () => void;
  onRefreshSync: () => void;
}

export function SettingsPanel({ settings, tools, syncStatus, onSave, onClose, onRefreshSync }: SettingsPanelProps) {
  const [localSettings, setLocalSettings] = useState<Settings>({ ...settings });
  const [defaultPaths, setDefaultPaths] = useState<Record<string, string[]>>({});
  const [saving, setSaving] = useState(false);
  const [version, setVersion] = useState("dev");

  useEffect(() => {
    GetDefaultScanPaths().then((paths) => {
      setDefaultPaths(paths ?? {});
    });
    GetVersion().then((v) => { if (v) setVersion(v); }).catch(() => {});
  }, []);

  // Sync local state when parent settings prop changes
  useEffect(() => {
    setLocalSettings({ ...settings });
  }, [settings]);

  const handleAddPath = async (toolId: ToolType) => {
    const path = await SelectDirectoryForTool();
    if (!path) return;

    const updated = { ...localSettings };
    const existing = updated.customPaths[toolId] ?? [];
    // Avoid duplicates
    if (existing.includes(path)) return;
    updated.customPaths = {
      ...updated.customPaths,
      [toolId]: [...existing, path],
    };
    setLocalSettings(updated);
    setSaving(true);
    await onSave(updated);
    setSaving(false);
  };

  const handleRemovePath = async (toolId: ToolType, pathToRemove: string) => {
    const updated = { ...localSettings };
    const existing = updated.customPaths[toolId] ?? [];
    updated.customPaths = {
      ...updated.customPaths,
      [toolId]: existing.filter((p) => p !== pathToRemove),
    };
    setLocalSettings(updated);
    setSaving(true);
    await onSave(updated);
    setSaving(false);
  };

  const handleThemeChange = async (theme: string) => {
    const updated = { ...localSettings, theme };
    setLocalSettings(updated);
    await onSave(updated);
  };

  const handleScanOnStartupChange = async (scanOnStartup: boolean) => {
    const updated = { ...localSettings, scanOnStartup };
    setLocalSettings(updated);
    await onSave(updated);
  };

  const handleRunInBackgroundChange = async (runInBackground: boolean) => {
    const updated = { ...localSettings, runInBackground };
    setLocalSettings(updated);
    await onSave(updated);
  };

  const truncatePath = (p: string, max: number = 55): string => {
    if (p.length <= max) return p;
    return "..." + p.slice(-(max - 3));
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%", overflow: "hidden" }}>
      {/* Header */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          padding: "12px 20px",
          borderBottom: "1px solid var(--border)",
          background: "var(--bg-2)",
          flexShrink: 0,
        }}
      >
        <span style={{ fontSize: 15, fontWeight: 600, color: "var(--text)", flex: 1 }}>
          Settings
        </span>
        {saving && (
          <span style={{ fontSize: 11, color: "var(--text-3)", marginRight: 12 }}>
            Saving...
          </span>
        )}
        <button
          onClick={onClose}
          style={{
            background: "transparent",
            border: "1px solid var(--border)",
            borderRadius: "var(--radius)",
            color: "var(--text-2)",
            padding: "3px 10px",
            fontSize: 12,
            cursor: "pointer",
            fontFamily: "var(--font-ui)",
          }}
        >
          {"\u2715"}
        </button>
      </div>

      {/* Scrollable content */}
      <div style={{ flex: 1, overflow: "auto", padding: "0 20px 20px" }}>

        {/* Section 1: Scan Paths */}
        <div style={{ marginTop: 20 }}>
          <div style={sectionHeaderStyle}>Scan Paths</div>
          <p style={sectionDescStyle}>
            Configure custom directories to scan for skill files. Default paths are shown for reference.
          </p>

          {tools.map((tool) => {
            const customPaths = localSettings.customPaths[tool.id] ?? [];
            const defaults = defaultPaths[tool.id] ?? [];

            return (
              <div key={tool.id} style={{ marginBottom: 16 }}>
                {/* Tool name with colored dot */}
                <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 6 }}>
                  <span
                    style={{
                      width: 8,
                      height: 8,
                      borderRadius: "50%",
                      background: tool.color,
                      flexShrink: 0,
                    }}
                  />
                  <span
                    style={{
                      fontSize: 13,
                      fontWeight: 600,
                      color: "var(--text)",
                      fontFamily: "var(--font-ui)",
                    }}
                  >
                    {tool.label}
                  </span>
                </div>

                {/* Default paths (read-only) */}
                {defaults.map((dp) => (
                  <div key={dp} style={pathRowStyle}>
                    <span
                      style={{
                        flex: 1,
                        fontFamily: "var(--font-mono)",
                        fontSize: 11,
                        color: "var(--text-3)",
                        overflow: "hidden",
                        textOverflow: "ellipsis",
                        whiteSpace: "nowrap",
                      }}
                      title={dp}
                    >
                      {truncatePath(dp)}
                    </span>
                    <span
                      style={{
                        fontSize: 9,
                        color: "var(--text-3)",
                        background: "var(--bg-2)",
                        padding: "1px 6px",
                        borderRadius: 3,
                        fontFamily: "var(--font-ui)",
                        fontWeight: 500,
                        textTransform: "uppercase",
                        letterSpacing: "0.04em",
                        flexShrink: 0,
                        border: "1px solid var(--border)",
                      }}
                    >
                      Default
                    </span>
                  </div>
                ))}

                {/* Custom paths */}
                {customPaths.map((cp) => (
                  <div key={cp} style={pathRowStyle}>
                    <span
                      style={{
                        flex: 1,
                        fontFamily: "var(--font-mono)",
                        fontSize: 11,
                        color: "var(--text-2)",
                        overflow: "hidden",
                        textOverflow: "ellipsis",
                        whiteSpace: "nowrap",
                      }}
                      title={cp}
                    >
                      {truncatePath(cp)}
                    </span>
                    <button
                      onClick={() => handleRemovePath(tool.id, cp)}
                      style={{
                        background: "transparent",
                        border: "none",
                        color: "var(--text-3)",
                        cursor: "pointer",
                        fontSize: 13,
                        padding: "0 4px",
                        lineHeight: 1,
                        flexShrink: 0,
                      }}
                      title="Remove path"
                      onMouseEnter={(e) => { e.currentTarget.style.color = "var(--red)"; }}
                      onMouseLeave={(e) => { e.currentTarget.style.color = "var(--text-3)"; }}
                    >
                      {"\u2715"}
                    </button>
                  </div>
                ))}

                {/* Add path button */}
                <button
                  onClick={() => handleAddPath(tool.id)}
                  style={{
                    background: "transparent",
                    border: "1px dashed var(--border-2)",
                    borderRadius: "var(--radius)",
                    color: "var(--text-3)",
                    padding: "4px 10px",
                    fontSize: 11,
                    cursor: "pointer",
                    fontFamily: "var(--font-ui)",
                    marginTop: 4,
                    transition: "border-color 0.15s, color 0.15s",
                  }}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.borderColor = "var(--accent)";
                    e.currentTarget.style.color = "var(--accent)";
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.borderColor = "var(--border-2)";
                    e.currentTarget.style.color = "var(--text-3)";
                  }}
                >
                  + Add path
                </button>
              </div>
            );
          })}
        </div>

        {/* Divider */}
        <div style={{ height: 1, background: "var(--border)", margin: "8px 0 20px" }} />

        {/* Section 2: General */}
        <div>
          <div style={sectionHeaderStyle}>General</div>

          {/* Theme toggle */}
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", padding: "10px 0" }}>
            <div>
              <div style={{ fontSize: 13, color: "var(--text)", fontFamily: "var(--font-ui)" }}>Theme</div>
              <div style={{ fontSize: 11, color: "var(--text-3)", marginTop: 2 }}>
                Visual appearance
              </div>
            </div>
            <div style={{ display: "flex", gap: 0, borderRadius: "var(--radius)", overflow: "hidden", border: "1px solid var(--border)" }}>
              <button
                onClick={() => handleThemeChange("dark")}
                style={{
                  background: localSettings.theme === "dark" ? "var(--accent)" : "var(--bg-3)",
                  border: "none",
                  color: localSettings.theme === "dark" ? "#fff" : "var(--text-3)",
                  padding: "4px 14px",
                  fontSize: 11,
                  cursor: "pointer",
                  fontFamily: "var(--font-ui)",
                  fontWeight: localSettings.theme === "dark" ? 600 : 400,
                }}
              >
                Dark
              </button>
              <button
                onClick={() => handleThemeChange("light")}
                style={{
                  background: localSettings.theme === "light" ? "var(--accent)" : "var(--bg-3)",
                  border: "none",
                  borderLeft: "1px solid var(--border)",
                  color: localSettings.theme === "light" ? "#fff" : "var(--text-3)",
                  padding: "4px 14px",
                  fontSize: 11,
                  cursor: "pointer",
                  fontFamily: "var(--font-ui)",
                  fontWeight: localSettings.theme === "light" ? 600 : 400,
                }}
              >
                Light
              </button>
            </div>
          </div>

          {/* Scan on startup */}
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", padding: "10px 0" }}>
            <div>
              <div style={{ fontSize: 13, color: "var(--text)", fontFamily: "var(--font-ui)" }}>Scan on startup</div>
              <div style={{ fontSize: 11, color: "var(--text-3)", marginTop: 2 }}>
                Automatically scan for skill files when the app launches
              </div>
            </div>
            <label
              style={{
                position: "relative",
                display: "inline-block",
                width: 36,
                height: 20,
                flexShrink: 0,
                cursor: "pointer",
              }}
            >
              <input
                type="checkbox"
                checked={localSettings.scanOnStartup}
                onChange={(e) => handleScanOnStartupChange(e.target.checked)}
                style={{ opacity: 0, width: 0, height: 0 }}
              />
              <span
                style={{
                  position: "absolute",
                  inset: 0,
                  borderRadius: 10,
                  background: localSettings.scanOnStartup ? "var(--accent)" : "var(--bg-3)",
                  border: "1px solid " + (localSettings.scanOnStartup ? "var(--accent)" : "var(--border-2)"),
                  transition: "background 0.2s, border-color 0.2s",
                }}
              />
              <span
                style={{
                  position: "absolute",
                  top: 3,
                  left: localSettings.scanOnStartup ? 19 : 3,
                  width: 14,
                  height: 14,
                  borderRadius: "50%",
                  background: "#fff",
                  transition: "left 0.2s",
                }}
              />
            </label>
          </div>

          {/* Close to background (Linux/Windows only) */}
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", padding: "10px 0" }}>
            <div>
              <div style={{ fontSize: 13, color: "var(--text)", fontFamily: "var(--font-ui)" }}>Close to background</div>
              <div style={{ fontSize: 11, color: "var(--text-3)", marginTop: 2 }}>
                Keep Agent Desk running when the window is closed (Linux/Windows)
              </div>
            </div>
            <label
              style={{
                position: "relative",
                display: "inline-block",
                width: 36,
                height: 20,
                flexShrink: 0,
                cursor: "pointer",
              }}
            >
              <input
                type="checkbox"
                checked={localSettings.runInBackground ?? false}
                onChange={(e) => handleRunInBackgroundChange(e.target.checked)}
                style={{ opacity: 0, width: 0, height: 0 }}
              />
              <span
                style={{
                  position: "absolute",
                  inset: 0,
                  borderRadius: 10,
                  background: (localSettings.runInBackground ?? false) ? "var(--accent)" : "var(--bg-3)",
                  border: "1px solid " + ((localSettings.runInBackground ?? false) ? "var(--accent)" : "var(--border-2)"),
                  transition: "background 0.2s, border-color 0.2s",
                }}
              />
              <span
                style={{
                  position: "absolute",
                  top: 3,
                  left: (localSettings.runInBackground ?? false) ? 19 : 3,
                  width: 14,
                  height: 14,
                  borderRadius: "50%",
                  background: "#fff",
                  transition: "left 0.2s",
                }}
              />
            </label>
          </div>
        </div>

        {/* Divider */}
        <div style={{ height: 1, background: "var(--border)", margin: "8px 0 20px" }} />

        {/* Section 4: Sync */}
        <div>
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <span style={sectionHeaderStyle}>Sync</span>
          </div>
          <p style={sectionDescStyle}>
            Sync your skill files across devices with encrypted peer-to-peer connections.
          </p>

          <SyncPanel syncStatus={syncStatus} onRefresh={onRefreshSync} />
        </div>

        {/* Divider */}
        <div style={{ height: 1, background: "var(--border)", margin: "8px 0 20px" }} />

        {/* Section 5: About */}
        <div>
          <div style={sectionHeaderStyle}>About</div>
          <div style={{ padding: "8px 0" }}>
            <div style={{ fontSize: 14, fontWeight: 600, color: "var(--text)", fontFamily: "var(--font-ui)" }}>
              Agent Desk {version}
            </div>
            <div style={{ fontSize: 12, color: "var(--text-3)", marginTop: 4 }}>
              AI skill file manager
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

const sectionHeaderStyle: React.CSSProperties = {
  fontSize: 11,
  fontWeight: 600,
  color: "var(--text-2)",
  textTransform: "uppercase",
  letterSpacing: "0.06em",
  marginBottom: 8,
  fontFamily: "var(--font-ui)",
};

const sectionDescStyle: React.CSSProperties = {
  fontSize: 12,
  color: "var(--text-3)",
  marginBottom: 16,
  lineHeight: 1.5,
};

const pathRowStyle: React.CSSProperties = {
  display: "flex",
  alignItems: "center",
  gap: 8,
  padding: "5px 10px",
  background: "var(--bg-3)",
  borderRadius: "var(--radius)",
  marginBottom: 4,
};
