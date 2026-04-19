import { useState, useCallback, useRef, useEffect, CSSProperties } from "react";
import { UpdateSkillMetadata } from "../../wailsjs/go/main/App";
import { Skill, ToolMeta, getToolMeta } from "../types";

interface MetadataPanelProps {
  skill: Skill;
  tools: ToolMeta[];
  onMetadataSaved: () => void;
}

// Keys that are always shown first and cannot be removed
const PINNED_KEYS = ["name", "description"];
// Keys shown as read-only badges (not editable via form)
const READONLY_KEYS = ["tool", "category"];

export function MetadataPanel({ skill, tools, onMetadataSaved }: MetadataPanelProps) {
  const [collapsed, setCollapsed] = useState(false);
  const [fields, setFields] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const [savedFlash, setSavedFlash] = useState(false);
  const [newKey, setNewKey] = useState("");
  const [addingField, setAddingField] = useState(false);
  const newKeyRef = useRef<HTMLInputElement>(null);

  const meta = getToolMeta(skill.tool, tools);
  const hasFrontmatter = Object.keys(skill.frontmatter ?? {}).length > 0;

  // Sync fields from skill.frontmatter when skill changes
  useEffect(() => {
    setFields({ ...skill.frontmatter });
  }, [skill.id, skill.frontmatter]);

  // Focus new key input when adding
  useEffect(() => {
    if (addingField && newKeyRef.current) {
      newKeyRef.current.focus();
    }
  }, [addingField]);

  const saveMetadata = useCallback(
    async (updatedFields: Record<string, string>) => {
      if (saving) return;
      setSaving(true);
      try {
        // Filter out readonly keys before sending
        const toSave: Record<string, string> = {};
        for (const [k, v] of Object.entries(updatedFields)) {
          if (!READONLY_KEYS.includes(k)) {
            toSave[k] = v;
          }
        }
        await UpdateSkillMetadata(skill.path, toSave);
        setSavedFlash(true);
        setTimeout(() => setSavedFlash(false), 1200);
        onMetadataSaved();
      } catch (err) {
        console.error("Metadata save failed:", err);
      } finally {
        setSaving(false);
      }
    },
    [skill.path, saving, onMetadataSaved],
  );

  const handleFieldBlur = (key: string, value: string) => {
    const prev = skill.frontmatter?.[key] ?? "";
    if (value === prev) return;
    const updated = { ...fields, [key]: value };
    setFields(updated);
    saveMetadata(updated);
  };

  const handleFieldKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.currentTarget.blur(); // blur triggers onBlur which calls handleFieldBlur
    }
  };

  const handleRemoveField = (key: string) => {
    const updated = { ...fields };
    delete updated[key];
    setFields(updated);
    saveMetadata(updated);
  };

  const handleAddField = () => {
    const trimmed = newKey.trim().toLowerCase().replace(/\s+/g, "_");
    if (!trimmed || fields[trimmed] !== undefined) {
      setNewKey("");
      setAddingField(false);
      return;
    }
    const updated = { ...fields, [trimmed]: "" };
    setFields(updated);
    setNewKey("");
    setAddingField(false);
    saveMetadata(updated);
  };

  const handleAddKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      handleAddField();
    } else if (e.key === "Escape") {
      setNewKey("");
      setAddingField(false);
    }
  };

  // Build ordered field list: pinned first, then the rest alphabetically
  const allKeys = Object.keys(fields);
  const pinnedPresent = PINNED_KEYS.filter((k) => allKeys.includes(k));
  const readonlyPresent = READONLY_KEYS.filter((k) => allKeys.includes(k));
  const otherKeys = allKeys
    .filter((k) => !PINNED_KEYS.includes(k) && !READONLY_KEYS.includes(k))
    .sort();

  // --- Styles ---

  const panelStyle: CSSProperties = {
    background: "var(--bg-2)",
    borderBottom: "1px solid var(--border)",
    flexShrink: 0,
    overflow: "hidden",
  };

  const headerRowStyle: CSSProperties = {
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    padding: "6px 16px",
    cursor: "pointer",
    userSelect: "none",
  };

  const headerLabelStyle: CSSProperties = {
    fontSize: 11,
    fontWeight: 600,
    fontFamily: "var(--font-mono)",
    color: "var(--text-2)",
    display: "flex",
    alignItems: "center",
    gap: 6,
  };

  const toggleStyle: CSSProperties = {
    fontSize: 10,
    color: "var(--text-3)",
    background: "transparent",
    border: "none",
    cursor: "pointer",
    fontFamily: "var(--font-mono)",
    padding: "2px 6px",
    borderRadius: "var(--radius)",
  };

  const fieldRowStyle: CSSProperties = {
    display: "flex",
    alignItems: "center",
    gap: 8,
    padding: "3px 16px",
  };

  const labelStyle: CSSProperties = {
    width: 100,
    fontSize: 11,
    fontFamily: "var(--font-mono)",
    color: "var(--text-3)",
    flexShrink: 0,
    textAlign: "right",
    paddingRight: 8,
  };

  const inputStyle: CSSProperties = {
    flex: 1,
    background: "var(--bg-3)",
    border: "1px solid var(--border)",
    borderRadius: "var(--radius)",
    color: "var(--text)",
    fontSize: 12,
    fontFamily: "var(--font-mono)",
    padding: "3px 8px",
    outline: "none",
  };

  const badgeStyle = (color: string): CSSProperties => ({
    display: "inline-flex",
    alignItems: "center",
    gap: 4,
    background: color + "20",
    color: color,
    padding: "1px 8px",
    borderRadius: "var(--radius)",
    fontSize: 11,
    fontWeight: 500,
    fontFamily: "var(--font-mono)",
  });

  const removeButtonStyle: CSSProperties = {
    background: "transparent",
    border: "none",
    color: "var(--text-3)",
    cursor: "pointer",
    fontSize: 12,
    fontFamily: "var(--font-mono)",
    padding: "2px 4px",
    borderRadius: "var(--radius)",
    lineHeight: 1,
    flexShrink: 0,
  };

  const addButtonStyle: CSSProperties = {
    background: "transparent",
    border: "1px solid var(--border)",
    borderRadius: "var(--radius)",
    color: "var(--text-3)",
    cursor: "pointer",
    fontSize: 11,
    fontFamily: "var(--font-mono)",
    padding: "2px 10px",
  };

  return (
    <div style={panelStyle}>
      {/* Header row -- always visible, click to toggle */}
      <div style={headerRowStyle} onClick={() => setCollapsed(!collapsed)}>
        <span style={headerLabelStyle}>
          <span style={{ fontSize: 9 }}>{collapsed ? "\u25b8" : "\u25be"}</span>
          Metadata
          {!hasFrontmatter && (
            <span style={{ fontSize: 10, color: "var(--text-3)", fontWeight: 400 }}>
              (no frontmatter)
            </span>
          )}
          {savedFlash && (
            <span style={{ fontSize: 10, color: "var(--green)", fontWeight: 500 }}>Saved</span>
          )}
        </span>
        <button
          style={toggleStyle}
          onClick={(e) => {
            e.stopPropagation();
            setCollapsed(!collapsed);
          }}
        >
          {collapsed ? "Show" : "Hide"}
        </button>
      </div>

      {/* Collapsible body */}
      {!collapsed && (
        <div style={{ paddingBottom: 8 }}>
          {/* Pinned fields: name, description */}
          {pinnedPresent.map((key) => (
            <FieldRow
              key={key}
              label={key}
              value={fields[key] ?? ""}
              onChange={(v) => setFields((f) => ({ ...f, [key]: v }))}
              onBlur={(v) => handleFieldBlur(key, v)}
              onKeyDown={(e) => handleFieldKeyDown(e)}
              inputStyle={inputStyle}
              labelStyle={labelStyle}
              fieldRowStyle={fieldRowStyle}
            />
          ))}

          {/* Name/Description placeholders when no frontmatter */}
          {!hasFrontmatter &&
            PINNED_KEYS.filter((k) => !pinnedPresent.includes(k)).map((key) => (
              <FieldRow
                key={key}
                label={key}
                value=""
                onChange={(v) => setFields((f) => ({ ...f, [key]: v }))}
                onBlur={(v) => handleFieldBlur(key, v)}
                onKeyDown={(e) => handleFieldKeyDown(e)}
                inputStyle={{ ...inputStyle, fontStyle: "italic", color: "var(--text-3)" }}
                labelStyle={labelStyle}
                fieldRowStyle={fieldRowStyle}
              />
            ))}

          {/* Readonly badges: tool, category */}
          {(readonlyPresent.length > 0 || skill.tool || skill.category) && (
            <div style={{ ...fieldRowStyle, gap: 12, paddingTop: 2, paddingBottom: 2, flexWrap: "wrap" }}>
              <span style={labelStyle}>{skill.tools && skill.tools.length > 1 ? "tools" : "tool"}</span>
              {(skill.tools && skill.tools.length > 1 ? skill.tools : [skill.tool]).map((t) => {
                const tMeta = getToolMeta(t, tools);
                return (
                  <span key={t} style={badgeStyle(tMeta.color)}>
                    <span>{tMeta.icon}</span>
                    {tMeta.label}
                  </span>
                );
              })}
              {skill.category && (
                <>
                  <span style={{ ...labelStyle, width: "auto", paddingLeft: 12 }}>category</span>
                  <span style={{
                    display: "inline-flex",
                    alignItems: "center",
                    gap: 4,
                    background: "var(--bg-3)",
                    color: "var(--text-2)",
                    padding: "1px 8px",
                    borderRadius: "var(--radius)",
                    fontSize: 11,
                    fontWeight: 500,
                    fontFamily: "var(--font-mono)",
                  }}>{skill.category}</span>
                </>
              )}
            </div>
          )}

          {/* Other dynamic fields */}
          {otherKeys.map((key) => (
            <div key={key} style={{ display: "flex", alignItems: "center" }}>
              <div style={{ flex: 1 }}>
                <FieldRow
                  label={key}
                  value={fields[key] ?? ""}
                  onChange={(v) => setFields((f) => ({ ...f, [key]: v }))}
                  onBlur={(v) => handleFieldBlur(key, v)}
                  onKeyDown={(e) => handleFieldKeyDown(e)}
                  inputStyle={inputStyle}
                  labelStyle={labelStyle}
                  fieldRowStyle={fieldRowStyle}
                />
              </div>
              <button
                style={removeButtonStyle}
                onClick={() => handleRemoveField(key)}
                title={`Remove "${key}"`}
              >
                x
              </button>
            </div>
          ))}

          {/* Add field row */}
          <div style={{ ...fieldRowStyle, paddingTop: 6 }}>
            <span style={labelStyle} />
            {addingField ? (
              <div style={{ display: "flex", gap: 6, alignItems: "center" }}>
                <input
                  ref={newKeyRef}
                  type="text"
                  value={newKey}
                  onChange={(e) => setNewKey(e.target.value)}
                  onKeyDown={handleAddKeyDown}
                  onBlur={handleAddField}
                  placeholder="field name"
                  style={{ ...inputStyle, width: 140 }}
                />
                <button
                  style={addButtonStyle}
                  onMouseDown={(e) => {
                    e.preventDefault();
                    handleAddField();
                  }}
                >
                  Add
                </button>
                <button
                  style={{ ...addButtonStyle, color: "var(--text-3)" }}
                  onMouseDown={(e) => {
                    e.preventDefault();
                    setNewKey("");
                    setAddingField(false);
                  }}
                >
                  Cancel
                </button>
              </div>
            ) : (
              <button style={addButtonStyle} onClick={() => setAddingField(true)}>
                + Add field
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// --- FieldRow sub-component ---

interface FieldRowProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  onBlur: (value: string) => void;
  onKeyDown: (e: React.KeyboardEvent<HTMLInputElement>) => void;
  inputStyle: CSSProperties;
  labelStyle: CSSProperties;
  fieldRowStyle: CSSProperties;
}

function FieldRow({ label, value, onChange, onBlur, onKeyDown, inputStyle, labelStyle, fieldRowStyle }: FieldRowProps) {
  const [localValue, setLocalValue] = useState(value);

  useEffect(() => {
    setLocalValue(value);
  }, [value]);

  return (
    <div style={fieldRowStyle}>
      <label style={labelStyle}>{label}</label>
      <input
        type="text"
        value={localValue}
        onChange={(e) => {
          setLocalValue(e.target.value);
          onChange(e.target.value);
        }}
        onBlur={() => onBlur(localValue)}
        onKeyDown={(e) => onKeyDown(e)}
        style={inputStyle}
      />
    </div>
  );
}
