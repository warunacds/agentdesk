import { useState } from "react";
import { CreateSkill, GetDefaultFilename, GetDefaultSubdir, SelectDirectory, CreateFromTemplate } from "../../wailsjs/go/main/App";
import { Skill, ToolMeta, ToolType, SkillTemplate } from "../types";

interface NewSkillModalProps {
  tools: ToolMeta[];
  templates: SkillTemplate[];
  onClose: () => void;
  onCreated: (skill: Skill) => void;
}

export function NewSkillModal({ tools, templates, onClose, onCreated }: NewSkillModalProps) {
  const [name, setName] = useState("");
  const [tool, setTool] = useState<ToolType>(tools[0]?.id ?? "claude-code");
  const [path, setPath] = useState("");
  const [error, setError] = useState("");
  const [creating, setCreating] = useState(false);
  const [selectedTemplateId, setSelectedTemplateId] = useState<string | null>(null);

  const handleBrowse = async () => {
    const dir = await SelectDirectory();
    if (!dir) return;

    const subdir = await GetDefaultSubdir(tool as any);
    const filename = await GetDefaultFilename(tool as any);
    const sep = "/";

    let fullPath = dir;
    if (subdir) {
      fullPath += sep + subdir;
    }
    fullPath += sep + (name ? name.replace(/[^a-zA-Z0-9._-]/g, "-").toLowerCase() + getExtension(tool) : filename);
    setPath(fullPath);
  };

  const handleToolChange = async (newTool: ToolType) => {
    setTool(newTool);
    if (path) {
      // Update filename portion of path
      const parts = path.split("/");
      const filename = await GetDefaultFilename(newTool as any);
      if (name) {
        parts[parts.length - 1] = name.replace(/[^a-zA-Z0-9._-]/g, "-").toLowerCase() + getExtension(newTool);
      } else {
        parts[parts.length - 1] = filename;
      }
      setPath(parts.join("/"));
    }
  };

  const getExtension = (t: ToolType): string => {
    switch (t) {
      case "cursor": return ".mdc";
      case "aider": return ".yml";
      default: return ".md";
    }
  };

  const handleSelectTemplate = (tmpl: SkillTemplate) => {
    setSelectedTemplateId(tmpl.id);
    if (!name) setName(tmpl.name);
    setTool(tmpl.tool as ToolType);
  };

  const handleCreate = async () => {
    if (!name.trim()) {
      setError("Name is required");
      return;
    }
    if (!path.trim()) {
      setError("Path is required. Click Browse to select a directory.");
      return;
    }
    setError("");
    setCreating(true);
    try {
      let skill;
      if (selectedTemplateId) {
        skill = await CreateFromTemplate(selectedTemplateId, name.trim(), path);
      } else {
        skill = await CreateSkill(path, tool as any, name.trim());
      }
      onCreated(skill as unknown as Skill);
    } catch (err: any) {
      setError(err?.message || "Failed to create skill");
    } finally {
      setCreating(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") onClose();
    if (e.key === "Enter" && !creating) handleCreate();
  };

  return (
    <div
      onKeyDown={handleKeyDown}
      style={{
        position: "fixed",
        inset: 0,
        zIndex: 100,
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        background: "rgba(0,0,0,0.6)",
        backdropFilter: "blur(4px)",
      }}
      onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}
    >
      <div
        style={{
          background: "var(--bg-2)",
          border: "1px solid var(--border)",
          borderRadius: 10,
          padding: 24,
          width: 620,
          maxWidth: "90vw",
          display: "flex",
          flexDirection: "column",
          gap: 16,
        }}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Title */}
        <div style={{ fontSize: 16, fontWeight: 600, color: "var(--text)" }}>
          New Skill
        </div>

        {/* Name input */}
        <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
          <label style={{ fontSize: 11, color: "var(--text-2)", fontWeight: 500 }}>Name</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. Code Review Guidelines"
            autoFocus
            style={{
              background: "var(--bg-3)",
              border: "1px solid var(--border)",
              borderRadius: "var(--radius)",
              color: "var(--text)",
              padding: "8px 12px",
              fontSize: 13,
              fontFamily: "var(--font-ui)",
              outline: "none",
            }}
          />
        </div>

        {/* Tool dropdown */}
        <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
          <label style={{ fontSize: 11, color: "var(--text-2)", fontWeight: 500 }}>Tool</label>
          <select
            value={tool}
            onChange={(e) => handleToolChange(e.target.value as ToolType)}
            style={{
              background: "var(--bg-3)",
              border: "1px solid var(--border)",
              borderRadius: "var(--radius)",
              color: "var(--text)",
              padding: "8px 12px",
              fontSize: 13,
              fontFamily: "var(--font-ui)",
              outline: "none",
              cursor: "pointer",
            }}
          >
            {tools.map((t) => (
              <option key={t.id} value={t.id}>
                {t.icon} {t.label}
              </option>
            ))}
          </select>
        </div>

        {/* Path input */}
        <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
          <label style={{ fontSize: 11, color: "var(--text-2)", fontWeight: 500 }}>Path</label>
          <div style={{ display: "flex", gap: 6 }}>
            <input
              type="text"
              value={path}
              onChange={(e) => setPath(e.target.value)}
              placeholder="Select a directory..."
              style={{
                flex: 1,
                background: "var(--bg-3)",
                border: "1px solid var(--border)",
                borderRadius: "var(--radius)",
                color: "var(--text)",
                padding: "8px 12px",
                fontSize: 12,
                fontFamily: "var(--font-mono)",
                outline: "none",
              }}
            />
            <button
              onClick={handleBrowse}
              style={{
                background: "var(--bg-3)",
                border: "1px solid var(--border)",
                borderRadius: "var(--radius)",
                color: "var(--text-2)",
                padding: "8px 14px",
                fontSize: 12,
                cursor: "pointer",
                fontFamily: "var(--font-ui)",
                whiteSpace: "nowrap",
              }}
            >
              Browse
            </button>
          </div>
        </div>

        {/* Templates */}
        {templates.length > 0 && (
          <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
            <label style={{ fontSize: 11, color: "var(--text-2)", fontWeight: 500 }}>Templates</label>
            <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 6, maxHeight: 160, overflowY: "auto" }}>
              {templates.map((tmpl) => {
                const tmplMeta = tools.find((t) => t.id === tmpl.tool);
                const isSelected = selectedTemplateId === tmpl.id;
                return (
                  <div
                    key={tmpl.id}
                    onClick={() => handleSelectTemplate(tmpl)}
                    style={{
                      background: isSelected ? "var(--accent-dim)" : "var(--bg-3)",
                      border: isSelected ? "1px solid var(--accent)" : "1px solid var(--border)",
                      borderRadius: "var(--radius)",
                      padding: "8px 10px",
                      cursor: "pointer",
                      transition: "border-color 0.1s, background 0.1s",
                    }}
                  >
                    <div style={{ fontSize: 12, fontWeight: 500, color: "var(--text)", marginBottom: 2, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                      {tmpl.name}
                    </div>
                    <div style={{ fontSize: 10, color: "var(--text-3)", marginBottom: 4, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                      {tmpl.description}
                    </div>
                    {tmplMeta && (
                      <span style={{
                        display: "inline-flex",
                        alignItems: "center",
                        gap: 3,
                        background: tmplMeta.color + "20",
                        color: tmplMeta.color,
                        padding: "1px 6px",
                        borderRadius: "var(--radius)",
                        fontSize: 9,
                        fontWeight: 600,
                        fontFamily: "var(--font-mono)",
                      }}>
                        <span>{tmplMeta.icon}</span>
                        <span>{tmplMeta.label}</span>
                      </span>
                    )}
                  </div>
                );
              })}
            </div>
            {selectedTemplateId && (
              <button
                onClick={() => setSelectedTemplateId(null)}
                style={{
                  background: "transparent",
                  border: "none",
                  color: "var(--text-3)",
                  fontSize: 11,
                  cursor: "pointer",
                  fontFamily: "var(--font-ui)",
                  textAlign: "left",
                  padding: 0,
                }}
              >
                Clear template selection
              </button>
            )}
          </div>
        )}

        {/* Error */}
        {error && (
          <div style={{ fontSize: 12, color: "var(--red)", padding: "4px 0" }}>
            {error}
          </div>
        )}

        {/* Actions */}
        <div style={{ display: "flex", justifyContent: "flex-end", gap: 8, marginTop: 4 }}>
          <button
            onClick={onClose}
            style={{
              background: "transparent",
              border: "1px solid var(--border)",
              borderRadius: "var(--radius)",
              color: "var(--text-2)",
              padding: "6px 16px",
              fontSize: 12,
              cursor: "pointer",
              fontFamily: "var(--font-ui)",
            }}
          >
            Cancel
          </button>
          <button
            onClick={handleCreate}
            disabled={creating}
            style={{
              background: "var(--accent)",
              border: "none",
              borderRadius: "var(--radius)",
              color: "#fff",
              padding: "6px 20px",
              fontSize: 12,
              fontWeight: 600,
              cursor: creating ? "default" : "pointer",
              fontFamily: "var(--font-ui)",
              opacity: creating ? 0.6 : 1,
            }}
          >
            {creating ? "Creating..." : "Create"}
          </button>
        </div>
      </div>
    </div>
  );
}
