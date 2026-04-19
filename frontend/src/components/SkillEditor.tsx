import { useRef, useEffect, useState, useCallback, useMemo } from "react";
import { EditorState } from "@codemirror/state";
import { EditorView, keymap, lineNumbers, highlightActiveLine, drawSelection } from "@codemirror/view";
import { defaultKeymap, history, historyKeymap, indentWithTab } from "@codemirror/commands";
import { markdown } from "@codemirror/lang-markdown";
import { json } from "@codemirror/lang-json";
import { yaml } from "@codemirror/lang-yaml";
import { python } from "@codemirror/lang-python";
import { syntaxHighlighting, defaultHighlightStyle, bracketMatching, foldGutter } from "@codemirror/language";
import { oneDark } from "@codemirror/theme-one-dark";
import { search, searchKeymap } from "@codemirror/search";
import { marked } from "marked";
import DOMPurify from "dompurify";
import { SaveSkill, RevealInFinder, GetSkills, ReadSkillFile } from "../../wailsjs/go/main/App";
import { Skill, ToolMeta, ValidationWarning, AuxFile, getToolMeta, formatBytes, formatDate } from "../types";
import { MetadataPanel } from "./MetadataPanel";

interface SkillEditorProps {
  skill: Skill;
  tools: ToolMeta[];
  theme: string;
  isTeamManaged?: boolean;
  validationWarnings?: ValidationWarning[];
  isDisabled?: boolean;
  onToggleDisabled?: () => Promise<void>;
  onSaved: (updated: Skill) => void;
}

const appDarkTheme = EditorView.theme({
  "&": {
    backgroundColor: "var(--bg)",
    color: "var(--text)",
    fontFamily: "var(--font-mono)",
    fontSize: "13px",
    height: "100%",
  },
  ".cm-content": {
    caretColor: "var(--accent)",
    padding: "12px 0",
  },
  "&.cm-focused .cm-cursor": {
    borderLeftColor: "var(--accent)",
  },
  "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection": {
    backgroundColor: "rgba(124,106,240,0.28)",
  },
  ".cm-activeLine": {
    backgroundColor: "rgba(255,255,255,0.02)",
  },
  ".cm-gutters": {
    backgroundColor: "var(--bg)",
    color: "var(--text-3)",
    border: "none",
    borderRight: "1px solid var(--border)",
  },
  ".cm-activeLineGutter": {
    backgroundColor: "rgba(255,255,255,0.02)",
  },
  ".cm-foldGutter": {
    color: "var(--text-3)",
  },
  ".cm-scroller": {
    overflow: "auto",
  },
});

const appLightTheme = EditorView.theme({
  "&": {
    backgroundColor: "#ffffff",
    color: "#1a1a2e",
    fontFamily: "var(--font-mono)",
    fontSize: "13px",
    height: "100%",
  },
  ".cm-content": {
    caretColor: "#6c5ce7",
    padding: "12px 0",
  },
  "&.cm-focused .cm-cursor": {
    borderLeftColor: "#6c5ce7",
  },
  "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection": {
    backgroundColor: "rgba(108,92,231,0.15)",
  },
  ".cm-activeLine": {
    backgroundColor: "rgba(0,0,0,0.03)",
  },
  ".cm-gutters": {
    backgroundColor: "#f7f7f8",
    color: "#8888a0",
    border: "none",
    borderRight: "1px solid #d8d9de",
  },
  ".cm-activeLineGutter": {
    backgroundColor: "#eeeff2",
  },
  ".cm-foldGutter": {
    color: "#8888a0",
  },
  ".cm-scroller": {
    overflow: "auto",
  },
}, { dark: false });

// --- File tree helpers (folder skills) ---

type TreeNode = {
  name: string;
  file?: AuxFile;
  children?: TreeNode[];
};

function buildTree(files: AuxFile[]): TreeNode[] {
  const root: TreeNode = { name: "", children: [] };
  for (const f of files) {
    const parts = f.relPath.split("/");
    let node = root;
    for (let i = 0; i < parts.length - 1; i++) {
      const segment = parts[i];
      let next = node.children?.find((c) => c.name === segment && c.children);
      if (!next) {
        next = { name: segment, children: [] };
        node.children = [...(node.children ?? []), next];
      }
      node = next;
    }
    node.children = [...(node.children ?? []), { name: parts[parts.length - 1], file: f }];
  }
  return root.children ?? [];
}

function FileTreeNode({ node, depth, currentPath, onSelect }: {
  node: TreeNode;
  depth: number;
  currentPath: string;
  onSelect: (file: AuxFile) => void;
}) {
  const [expanded, setExpanded] = useState(true);
  if (node.file) {
    const isActive = currentPath === node.file.path;
    return (
      <div
        onClick={() => onSelect(node.file!)}
        style={{
          cursor: "pointer",
          padding: "2px 4px",
          paddingLeft: 4 + depth * 12,
          fontSize: 11,
          color: isActive ? "var(--accent)" : "var(--text-2)",
          fontWeight: isActive ? 600 : 400,
          fontFamily: "var(--font-ui)",
        }}
      >
        {"\uD83D\uDCC4"} {node.name}
      </div>
    );
  }
  return (
    <div>
      <div
        onClick={() => setExpanded(!expanded)}
        style={{
          cursor: "pointer",
          padding: "2px 4px",
          paddingLeft: 4 + depth * 12,
          fontSize: 11,
          color: "var(--text-2)",
          fontFamily: "var(--font-ui)",
        }}
      >
        {expanded ? "\u25BE" : "\u25B8"} {"\uD83D\uDCC1"} {node.name}
      </div>
      {expanded && node.children && node.children.map((child) => (
        <FileTreeNode
          key={child.name + depth}
          node={child}
          depth={depth + 1}
          currentPath={currentPath}
          onSelect={onSelect}
        />
      ))}
    </div>
  );
}

export function SkillEditor({ skill, tools, theme, isTeamManaged, validationWarnings, isDisabled, onToggleDisabled, onSaved }: SkillEditorProps) {
  const editorRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [savedFlash, setSavedFlash] = useState(false);
  const [autoSavedFlash, setAutoSavedFlash] = useState(false);
  const [lineCount, setLineCount] = useState(0);
  const [charCount, setCharCount] = useState(0);
  const [wordCount, setWordCount] = useState(0);
  const [showPreview, setShowPreview] = useState<boolean>(() => isMarkdownFile(skill.path));
  const [currentFilePath, setCurrentFilePath] = useState<string>(skill.path);
  const [currentFileContent, setCurrentFileContent] = useState<string>(skill.content);
  const [currentFileLanguage, setCurrentFileLanguage] = useState<string>("markdown");
  const [showFiles, setShowFiles] = useState(false);
  const contentRef = useRef(skill.content);
  const originalContentRef = useRef(skill.content);
  const autoSaveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const meta = getToolMeta(skill.tool, tools);

  // Reset aux-file state when the skill changes
  useEffect(() => {
    setCurrentFilePath(skill.path);
    setCurrentFileContent(skill.content);
    setCurrentFileLanguage("markdown");
    setShowPreview(isMarkdownFile(skill.path));
  }, [skill.id, skill.path, skill.content]);

  const handleSelectAuxFile = useCallback(async (file: AuxFile) => {
    try {
      const result = await ReadSkillFile(file.path);
      if (!result) return;
      setCurrentFilePath(file.path);
      setCurrentFileContent(result.content);
      setCurrentFileLanguage(result.language);
      setShowPreview(isMarkdownFile(file.path));
    } catch (err) {
      console.error("Failed to read aux file:", err);
    }
  }, []);

  const handleReturnToMain = useCallback(() => {
    setCurrentFilePath(skill.path);
    setCurrentFileContent(skill.content);
    setCurrentFileLanguage("markdown");
    setShowPreview(isMarkdownFile(skill.path));
  }, [skill.path, skill.content]);

  // After metadata is saved via the form, re-read the file to sync the editor
  const handleMetadataSaved = useCallback(async () => {
    try {
      const allSkills = (await GetSkills()) as unknown as Skill[];
      const refreshed = allSkills.find((s) => s.path === skill.path);
      if (refreshed && viewRef.current) {
        const view = viewRef.current;
        const currentContent = view.state.doc.toString();
        if (refreshed.content !== currentContent) {
          view.dispatch({
            changes: { from: 0, to: currentContent.length, insert: refreshed.content },
          });
          contentRef.current = refreshed.content;
          setDirty(false);
        }
        onSaved({ ...refreshed });
      }
    } catch (err) {
      console.error("Failed to refresh after metadata save:", err);
    }
  }, [skill.path, onSaved]);

  // Internal save without UI flash -- used by auto-save
  const doSaveInternal = useCallback(async (): Promise<boolean> => {
    if (saving) return false;
    const content = contentRef.current;
    setSaving(true);
    try {
      await SaveSkill(currentFilePath, content);
      setDirty(false);
      if (currentFilePath === skill.path) {
        onSaved({ ...skill, content, size: new Blob([content]).size, modified: Math.floor(Date.now() / 1000) });
      } else {
        setCurrentFileContent(content);
      }
      return true;
    } catch (err) {
      console.error("Save failed:", err);
      return false;
    } finally {
      setSaving(false);
    }
  }, [skill, saving, onSaved, currentFilePath]);

  const doSave = useCallback(async () => {
    const ok = await doSaveInternal();
    if (ok) {
      setSavedFlash(true);
      setTimeout(() => setSavedFlash(false), 1500);
    }
  }, [doSaveInternal]);

  // Auto-save: debounce saves after 2s of inactivity
  const scheduleAutoSave = useCallback(() => {
    if (isTeamManaged) return;
    if (autoSaveTimerRef.current) clearTimeout(autoSaveTimerRef.current);
    autoSaveTimerRef.current = setTimeout(async () => {
      const current = contentRef.current;
      if (current !== originalContentRef.current) {
        const ok = await doSaveInternal();
        if (ok) {
          setAutoSavedFlash(true);
          setTimeout(() => setAutoSavedFlash(false), 1500);
          originalContentRef.current = current;
        }
      }
    }, 2000);
  }, [doSaveInternal, isTeamManaged]);

  // Clean up auto-save timer on unmount
  useEffect(() => {
    return () => {
      if (autoSaveTimerRef.current) clearTimeout(autoSaveTimerRef.current);
    };
  }, []);

  useEffect(() => {
    if (!editorRef.current) return;

    const countWords = (text: string) => text.split(/\s+/).filter(Boolean).length;

    const saveKeymap = keymap.of([
      {
        key: "Mod-s",
        run: () => {
          doSave();
          return true;
        },
      },
    ]);

    const startState = EditorState.create({
      doc: contentRef.current || currentFileContent,
      extensions: [
        saveKeymap,
        lineNumbers(),
        highlightActiveLine(),
        drawSelection(),
        history(),
        foldGutter(),
        bracketMatching(),
        syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
        getLanguageForName(currentFileLanguage),
        search(),
        ...(theme === "light" ? [appLightTheme] : [oneDark, appDarkTheme]),
        EditorView.lineWrapping,
        ...(isTeamManaged ? [EditorState.readOnly.of(true), EditorView.editable.of(false)] : []),
        keymap.of([...defaultKeymap, ...historyKeymap, ...searchKeymap, indentWithTab]),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            const doc = update.state.doc;
            const text = doc.toString();
            contentRef.current = text;
            setDirty(true);
            setLineCount(doc.lines);
            setCharCount(doc.length);
            setWordCount(countWords(text));
            scheduleAutoSave();
          }
        }),
      ],
    });

    const view = new EditorView({
      state: startState,
      parent: editorRef.current,
    });

    viewRef.current = view;
    setLineCount(startState.doc.lines);
    setCharCount(startState.doc.length);
    setWordCount(startState.doc.toString().split(/\s+/).filter(Boolean).length);
    contentRef.current = currentFileContent;
    originalContentRef.current = currentFileContent;
    setDirty(false);

    return () => {
      view.destroy();
      if (autoSaveTimerRef.current) clearTimeout(autoSaveTimerRef.current);
    };
  }, [skill.id, currentFilePath, currentFileLanguage, showPreview]); // eslint-disable-line react-hooks/exhaustive-deps

  // Sync editor doc when currentFileContent changes (e.g. after aux-file save)
  useEffect(() => {
    if (!viewRef.current) return;
    const view = viewRef.current;
    const currentDoc = view.state.doc.toString();
    if (currentDoc !== currentFileContent) {
      view.dispatch({
        changes: { from: 0, to: currentDoc.length, insert: currentFileContent },
      });
    }
  }, [currentFileContent]);

  const handleReveal = () => {
    RevealInFinder(currentFilePath);
  };

  const truncatedPath = currentFilePath.length > 50
    ? "..." + currentFilePath.slice(-47)
    : currentFilePath;

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%", overflow: "hidden" }}>
      {/* Header bar */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 10,
          padding: "8px 16px",
          borderBottom: "1px solid var(--border)",
          background: "var(--bg-2)",
          flexShrink: 0,
        }}
      >
        {/* Tool badge(s) — show all tools when skill is shared via symlinks */}
        {(skill.tools && skill.tools.length > 1 ? skill.tools : [skill.tool]).map((t) => {
          const badgeMeta = getToolMeta(t, tools);
          return (
            <span
              key={t}
              style={{
                display: "inline-flex",
                alignItems: "center",
                gap: 5,
                background: badgeMeta.color + "20",
                color: badgeMeta.color,
                padding: "2px 8px",
                borderRadius: "var(--radius)",
                fontSize: 11,
                fontWeight: 600,
                fontFamily: "var(--font-mono)",
              }}
            >
              <span>{badgeMeta.icon}</span>
              <span>{badgeMeta.label}</span>
            </span>
          );
        })}

        {/* Skill name (or breadcrumb when viewing aux file) */}
        <span style={{ fontSize: 13, fontWeight: 500, color: "var(--text)" }}>
          {currentFilePath === skill.path ? skill.name : (
            <>
              <span
                onClick={handleReturnToMain}
                style={{ cursor: "pointer", color: "var(--text-2)" }}
              >
                {skill.name}
              </span>
              <span style={{ color: "var(--text-3)", margin: "0 6px" }}>{"\u203A"}</span>
              <span>{skill.auxiliaryFiles?.find((f) => f.path === currentFilePath)?.relPath ?? ""}</span>
            </>
          )}
        </span>

        {/* Dirty indicator */}
        {dirty && (
          <span style={{ width: 6, height: 6, borderRadius: "50%", background: "var(--yellow)", flexShrink: 0 }} title="Unsaved changes" />
        )}

        {/* Saved flash */}
        {savedFlash && (
          <span style={{ fontSize: 11, color: "var(--green)", fontWeight: 500, transition: "opacity 0.3s" }}>
            Saved
          </span>
        )}

        {/* Auto-saved flash */}
        {autoSavedFlash && !savedFlash && (
          <span style={{ fontSize: 11, color: "var(--green)", fontWeight: 500, transition: "opacity 0.3s" }}>
            Auto-saved
          </span>
        )}

        {/* Validation warnings */}
        {validationWarnings && validationWarnings.length > 0 && (
          <span
            title={validationWarnings.map((w) => w.message).join("\n")}
            style={{
              display: "inline-flex",
              alignItems: "center",
              gap: 4,
              fontSize: 11,
              color: "var(--yellow)",
              fontWeight: 500,
              cursor: "default",
            }}
          >
            <span style={{ width: 6, height: 6, borderRadius: "50%", background: "var(--yellow)", flexShrink: 0 }} />
            {validationWarnings.length} warning{validationWarnings.length !== 1 ? "s" : ""}
          </span>
        )}

        <div style={{ flex: 1 }} />

        {/* Preview toggle — markdown files only */}
        {isMarkdownFile(currentFilePath) && (
          <div style={{ display: "flex", gap: 0, borderRadius: "var(--radius)", overflow: "hidden", border: "1px solid var(--border)" }}>
            <button
              onClick={() => setShowPreview(false)}
              style={{
                background: !showPreview ? "var(--accent)" : "transparent",
                border: "none",
                color: !showPreview ? "#fff" : "var(--text-3)",
                padding: "3px 10px",
                fontSize: 11,
                fontWeight: !showPreview ? 600 : 400,
                cursor: "pointer",
                fontFamily: "var(--font-ui)",
              }}
            >
              Edit
            </button>
            <button
              onClick={() => setShowPreview(true)}
              style={{
                background: showPreview ? "var(--accent)" : "transparent",
                border: "none",
                borderLeft: "1px solid var(--border)",
                color: showPreview ? "#fff" : "var(--text-3)",
                padding: "3px 10px",
                fontSize: 11,
                fontWeight: showPreview ? 600 : 400,
                cursor: "pointer",
                fontFamily: "var(--font-ui)",
              }}
            >
              Preview
            </button>
          </div>
        )}

        {/* Reveal button */}
        <button
          onClick={handleReveal}
          style={{
            background: "transparent",
            border: "1px solid var(--border)",
            borderRadius: "var(--radius)",
            color: "var(--text-2)",
            padding: "3px 10px",
            fontSize: 11,
            cursor: "pointer",
            fontFamily: "var(--font-ui)",
          }}
        >
          Reveal
        </button>

        {/* Disable / Enable toggle */}
        {onToggleDisabled && (
          <button
            onClick={onToggleDisabled}
            style={{
              background: isDisabled ? "var(--yellow)" : "transparent",
              border: "1px solid var(--border)",
              borderRadius: "var(--radius)",
              color: isDisabled ? "#1a1a26" : "var(--text-2)",
              padding: "3px 10px",
              fontSize: 11,
              fontWeight: isDisabled ? 600 : 400,
              cursor: "pointer",
              fontFamily: "var(--font-ui)",
            }}
            title={isDisabled ? "This skill is disabled — click to enable" : "Disable this skill for its tool"}
          >
            {isDisabled ? "Disabled" : "Disable"}
          </button>
        )}

        {/* Save button — hidden for team-managed files */}
        {!isTeamManaged && (
          <button
            onClick={doSave}
            disabled={!dirty || saving}
            style={{
              background: dirty ? "var(--accent)" : "var(--bg-3)",
              border: "none",
              borderRadius: "var(--radius)",
              color: dirty ? "#fff" : "var(--text-3)",
              padding: "3px 14px",
              fontSize: 11,
              fontWeight: 600,
              cursor: dirty ? "pointer" : "default",
              fontFamily: "var(--font-ui)",
              opacity: dirty ? 1 : 0.5,
            }}
          >
            {saving ? "Saving..." : "Save"}
          </button>
        )}
      </div>

      {/* Team-managed banner */}
      {isTeamManaged && (
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 8,
            padding: "6px 16px",
            background: "rgba(124,106,240,0.08)",
            borderBottom: "1px solid rgba(124,106,240,0.2)",
            fontSize: 12,
            color: "var(--accent)",
            fontFamily: "var(--font-ui)",
            flexShrink: 0,
          }}
        >
          <span style={{ fontSize: 13 }}>{"\uD83D\uDD12"}</span>
          <span>This file is managed by your team. Changes will be overwritten on next sync.</span>
        </div>
      )}

      {/* Files tree — only for folder skills with aux files */}
      {skill.directory && skill.auxiliaryFiles && skill.auxiliaryFiles.length > 0 && (
        <div style={{
          borderBottom: "1px solid var(--border)",
          background: "var(--bg-2)",
          padding: "6px 12px",
          fontSize: 12,
          fontFamily: "var(--font-ui)",
          flexShrink: 0,
        }}>
          <button
            onClick={() => setShowFiles(!showFiles)}
            style={{
              background: "transparent",
              border: "none",
              color: "var(--text-2)",
              cursor: "pointer",
              fontSize: 11,
              fontWeight: 600,
              padding: 0,
              fontFamily: "var(--font-ui)",
            }}
          >
            {showFiles ? "\u25BE" : "\u25B8"} Files ({skill.auxiliaryFiles.length})
          </button>
          {showFiles && (
            <div style={{ marginTop: 6 }}>
              <div
                onClick={handleReturnToMain}
                style={{
                  cursor: "pointer",
                  padding: "2px 4px",
                  fontSize: 11,
                  color: currentFilePath === skill.path ? "var(--accent)" : "var(--text-2)",
                  fontWeight: currentFilePath === skill.path ? 600 : 400,
                }}
              >
                {"\uD83D\uDCC4"} SKILL.md
              </div>
              {buildTree(skill.auxiliaryFiles).map((node) => (
                <FileTreeNode
                  key={node.name}
                  node={node}
                  depth={0}
                  currentPath={currentFilePath}
                  onSelect={handleSelectAuxFile}
                />
              ))}
            </div>
          )}
        </div>
      )}

      {/* Metadata panel — only for markdown files with frontmatter (main file only) */}
      {isMarkdownFile(currentFilePath) && currentFilePath === skill.path && <MetadataPanel skill={skill} tools={tools} onMetadataSaved={handleMetadataSaved} />}

      {/* Markdown toolbar — only in edit mode */}
      {isMarkdownFile(currentFilePath) && !showPreview && <MarkdownToolbar viewRef={viewRef} />}

      {/* Preview or Editor */}
      {showPreview && isMarkdownFile(currentFilePath) ? (
        <MarkdownPreview content={contentRef.current} theme={theme} />
      ) : (
        <div ref={editorRef} style={{ flex: 1, overflow: "hidden" }} />
      )}

      {/* Status bar */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 16,
          padding: "4px 16px",
          borderTop: "1px solid var(--border)",
          background: "var(--bg-2)",
          fontSize: 11,
          color: "var(--text-3)",
          fontFamily: "var(--font-mono)",
          flexShrink: 0,
        }}
      >
        <span style={{ display: "flex", alignItems: "center", gap: 4 }}>
          <span style={{ width: 6, height: 6, borderRadius: "50%", background: meta.color }} />
          {meta.label}
        </span>
        <span style={{ background: "var(--bg-3)", padding: "1px 6px", borderRadius: 3, fontSize: 10 }}>{getFileTypeLabel(currentFilePath)}</span>
        <span>{lineCount} lines</span>
        <span>{charCount} chars</span>
        <span>{wordCount} words</span>
        <span>{formatBytes(currentFilePath === skill.path ? skill.size : new Blob([currentFileContent]).size)}</span>
        <div style={{ flex: 1 }} />
        <span title={currentFilePath} style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap", maxWidth: 300 }}>
          {truncatedPath}
        </span>
        <span>{formatDate(skill.modified)}</span>
      </div>
    </div>
  );
}

// --- File type detection ---

function getFileExtension(path: string): string {
  const lastDot = path.lastIndexOf(".");
  if (lastDot < 0) return "";
  return path.substring(lastDot).toLowerCase();
}

function getFileType(path: string): "markdown" | "json" | "yaml" | "toml" | "text" {
  const ext = getFileExtension(path);
  const filename = path.split("/").pop()?.toLowerCase() || "";
  switch (ext) {
    case ".json": return "json";
    case ".yaml": case ".yml": return "yaml";
    case ".toml": return "toml";
    case ".md": case ".markdown": case ".mdc": case ".mdx": return "markdown";
    default:
      // Check filenames without extensions
      if (filename === ".cursorrules" || filename === ".cursorignore" || filename === ".windsurfrules") {
        return "text";
      }
      return "markdown";
  }
}

function getLanguageExtension(path: string) {
  const fileType = getFileType(path);
  switch (fileType) {
    case "json": return json();
    case "yaml": return yaml();
    case "markdown": return markdown();
    default: return markdown(); // fallback
  }
}

function getLanguageForName(lang: string) {
  switch (lang) {
    case "json": return json();
    case "yaml": return yaml();
    case "python": return python();
    case "markdown": return markdown();
    default: return markdown();
  }
}

function isMarkdownFile(path: string): boolean {
  return getFileType(path) === "markdown";
}

function getFileTypeLabel(path: string): string {
  const ft = getFileType(path);
  switch (ft) {
    case "json": return "JSON";
    case "yaml": return "YAML";
    case "toml": return "TOML";
    case "markdown": return "Markdown";
    default: return "Text";
  }
}

// --- Markdown Toolbar ---

interface MarkdownToolbarProps {
  viewRef: React.RefObject<EditorView | null>;
}

function MarkdownToolbar({ viewRef }: MarkdownToolbarProps) {
  const wrapSelection = (before: string, after: string) => {
    const view = viewRef.current;
    if (!view) return;
    const { from, to } = view.state.selection.main;
    const selected = view.state.sliceDoc(from, to);
    const replacement = before + (selected || "text") + after;
    view.dispatch({
      changes: { from, to, insert: replacement },
      selection: { anchor: from + before.length, head: from + before.length + (selected || "text").length },
    });
    view.focus();
  };

  const insertAtLineStart = (prefix: string) => {
    const view = viewRef.current;
    if (!view) return;
    const { from } = view.state.selection.main;
    const line = view.state.doc.lineAt(from);
    view.dispatch({
      changes: { from: line.from, to: line.from, insert: prefix },
    });
    view.focus();
  };

  const insertBlock = (text: string) => {
    const view = viewRef.current;
    if (!view) return;
    const { from, to } = view.state.selection.main;
    view.dispatch({
      changes: { from, to, insert: text },
      selection: { anchor: from + text.indexOf("text"), head: from + text.indexOf("text") + 4 },
    });
    view.focus();
  };

  const buttons: { label: string; title: string; action: () => void }[] = [
    { label: "B", title: "Bold (Cmd+B in editor)", action: () => wrapSelection("**", "**") },
    { label: "I", title: "Italic", action: () => wrapSelection("*", "*") },
    { label: "S", title: "Strikethrough", action: () => wrapSelection("~~", "~~") },
    { label: "`", title: "Inline code", action: () => wrapSelection("`", "`") },
    { label: "H1", title: "Heading 1", action: () => insertAtLineStart("# ") },
    { label: "H2", title: "Heading 2", action: () => insertAtLineStart("## ") },
    { label: "H3", title: "Heading 3", action: () => insertAtLineStart("### ") },
    { label: "\u2014", title: "Horizontal rule", action: () => insertBlock("\n---\n") },
    { label: "\u2022", title: "Bullet list", action: () => insertAtLineStart("- ") },
    { label: "1.", title: "Numbered list", action: () => insertAtLineStart("1. ") },
    { label: "\u2610", title: "Checkbox", action: () => insertAtLineStart("- [ ] ") },
    { label: ">", title: "Blockquote", action: () => insertAtLineStart("> ") },
    { label: "{ }", title: "Code block", action: () => insertBlock("\n```\ntext\n```\n") },
    { label: "\uD83D\uDD17", title: "Link", action: () => wrapSelection("[", "](url)") },
  ];

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 2,
        padding: "3px 12px",
        borderBottom: "1px solid var(--border)",
        background: "var(--bg-2)",
        flexShrink: 0,
        flexWrap: "wrap",
      }}
    >
      {buttons.map((btn, i) => (
        <button
          key={i}
          onClick={btn.action}
          title={btn.title}
          style={mdToolbarBtnStyle}
          onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-hover)"; e.currentTarget.style.color = "var(--text)"; }}
          onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; e.currentTarget.style.color = "var(--text-3)"; }}
        >
          {btn.label}
        </button>
      ))}
    </div>
  );
}

// --- Markdown Preview ---

function stripFrontmatter(content: string): string {
  const match = content.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n?/);
  return match ? content.slice(match[0].length) : content;
}

function MarkdownPreview({ content, theme }: { content: string; theme: string }) {
  const html = useMemo(() => {
    const body = stripFrontmatter(content);
    const raw = marked.parse(body, { async: false }) as string;
    return DOMPurify.sanitize(raw);
  }, [content]);

  const isDark = theme !== "light";

  return (
    <div
      style={{
        flex: 1,
        overflow: "auto",
        padding: "24px 32px",
        fontFamily: "var(--font-ui)",
        fontSize: 14,
        lineHeight: 1.7,
        color: isDark ? "#e8e8f0" : "#1a1a2e",
        background: isDark ? "var(--bg)" : "#ffffff",
      }}
    >
      <div
        dangerouslySetInnerHTML={{ __html: html }}
        style={{ maxWidth: 720 }}
        className="md-preview"
      />
      <style>{`
        .md-preview h1 { font-size: 1.8em; font-weight: 700; margin: 1.2em 0 0.6em; border-bottom: 1px solid ${isDark ? "rgba(255,255,255,0.08)" : "rgba(0,0,0,0.08)"}; padding-bottom: 0.3em; }
        .md-preview h2 { font-size: 1.4em; font-weight: 600; margin: 1em 0 0.5em; border-bottom: 1px solid ${isDark ? "rgba(255,255,255,0.05)" : "rgba(0,0,0,0.05)"}; padding-bottom: 0.25em; }
        .md-preview h3 { font-size: 1.15em; font-weight: 600; margin: 0.8em 0 0.4em; }
        .md-preview p { margin: 0.6em 0; }
        .md-preview a { color: ${isDark ? "#9580f5" : "#6c5ce7"}; text-decoration: none; }
        .md-preview a:hover { text-decoration: underline; }
        .md-preview code { font-family: var(--font-mono); font-size: 0.88em; background: ${isDark ? "rgba(255,255,255,0.06)" : "rgba(0,0,0,0.06)"}; padding: 2px 5px; border-radius: 3px; }
        .md-preview pre { background: ${isDark ? "#1a1a26" : "#f5f5f7"}; border: 1px solid ${isDark ? "rgba(255,255,255,0.08)" : "rgba(0,0,0,0.08)"}; border-radius: 6px; padding: 14px 16px; overflow-x: auto; margin: 0.8em 0; }
        .md-preview pre code { background: none; padding: 0; font-size: 13px; }
        .md-preview blockquote { border-left: 3px solid ${isDark ? "rgba(149,128,245,0.4)" : "rgba(108,92,231,0.3)"}; margin: 0.6em 0; padding: 0.4em 0 0.4em 16px; color: ${isDark ? "#9090a8" : "#666680"}; }
        .md-preview ul, .md-preview ol { padding-left: 24px; margin: 0.5em 0; }
        .md-preview li { margin: 0.25em 0; }
        .md-preview hr { border: none; border-top: 1px solid ${isDark ? "rgba(255,255,255,0.08)" : "rgba(0,0,0,0.08)"}; margin: 1.5em 0; }
        .md-preview table { border-collapse: collapse; width: 100%; margin: 0.8em 0; }
        .md-preview th, .md-preview td { border: 1px solid ${isDark ? "rgba(255,255,255,0.1)" : "rgba(0,0,0,0.1)"}; padding: 8px 12px; text-align: left; }
        .md-preview th { font-weight: 600; background: ${isDark ? "rgba(255,255,255,0.03)" : "rgba(0,0,0,0.03)"}; }
        .md-preview img { max-width: 100%; border-radius: 6px; }
        .md-preview input[type="checkbox"] { margin-right: 6px; }
      `}</style>
    </div>
  );
}

const mdToolbarBtnStyle: React.CSSProperties = {
  background: "transparent",
  border: "none",
  borderRadius: "var(--radius)",
  color: "var(--text-3)",
  padding: "2px 6px",
  fontSize: 12,
  fontWeight: 600,
  fontFamily: "var(--font-mono)",
  cursor: "pointer",
  minWidth: 24,
  textAlign: "center",
  transition: "background 0.1s, color 0.1s",
};
