import { useState, CSSProperties } from "react";
import { SyncConflict } from "../types";

interface SyncConflictDialogProps {
  conflicts: SyncConflict[];
  onResolve: (path: string, resolution: string) => void;
  onClose: () => void;
}

export function SyncConflictDialog({ conflicts, onResolve, onClose }: SyncConflictDialogProps) {
  const [activeIndex, setActiveIndex] = useState(0);
  const [resolving, setResolving] = useState(false);

  if (conflicts.length === 0) return null;

  const conflict = conflicts[Math.min(activeIndex, conflicts.length - 1)];
  const fileName = conflict.path.split("/").pop() || conflict.path;

  const handleResolve = async (resolution: string) => {
    setResolving(true);
    try {
      await onResolve(conflict.path, resolution);
      // Move to next conflict or close if none remain
      if (conflicts.length <= 1) {
        onClose();
      } else if (activeIndex >= conflicts.length - 1) {
        setActiveIndex(Math.max(0, conflicts.length - 2));
      }
    } finally {
      setResolving(false);
    }
  };

  const truncateContent = (content: string, maxLen: number = 500): string => {
    if (!content) return "(empty)";
    if (content.length <= maxLen) return content;
    return content.slice(0, maxLen) + "\n\u2026 (truncated)";
  };

  return (
    <div style={overlayStyle}>
      <div style={dialogStyle}>
        {/* Header */}
        <div style={headerStyle}>
          <span style={{ fontSize: 14, fontWeight: 600, color: "var(--text)" }}>
            Sync Conflict: {fileName}
          </span>
          <button onClick={onClose} style={closeButtonStyle}>
            {"\u2715"}
          </button>
        </div>

        {/* Conflict nav (if multiple) */}
        {conflicts.length > 1 && (
          <div style={navStyle}>
            {conflicts.map((c, i) => (
              <button
                key={c.path}
                onClick={() => setActiveIndex(i)}
                style={{
                  ...navButtonStyle,
                  background: i === activeIndex ? "var(--accent-dim)" : "transparent",
                  color: i === activeIndex ? "var(--accent)" : "var(--text-3)",
                  borderColor: i === activeIndex ? "var(--accent)" : "var(--border)",
                }}
              >
                {c.path.split("/").pop()}
              </button>
            ))}
          </div>
        )}

        {/* File path */}
        <div style={pathStyle}>
          {conflict.path}
        </div>

        {/* Two-pane diff view */}
        <div style={panesContainerStyle}>
          <div style={paneStyle}>
            <div style={paneLabelStyle}>Local version</div>
            <pre style={paneContentStyle}>
              {truncateContent(conflict.localContent)}
            </pre>
          </div>
          <div style={{ width: 1, background: "var(--border)", flexShrink: 0 }} />
          <div style={paneStyle}>
            <div style={paneLabelStyle}>Remote version</div>
            <pre style={paneContentStyle}>
              {truncateContent(conflict.remoteContent)}
            </pre>
          </div>
        </div>

        {/* Resolution buttons */}
        <div style={actionsStyle}>
          <button
            onClick={() => handleResolve("keep-local")}
            disabled={resolving}
            style={actionButtonStyle}
          >
            Keep Local
          </button>
          <button
            onClick={() => handleResolve("keep-remote")}
            disabled={resolving}
            style={actionButtonStyle}
          >
            Keep Remote
          </button>
          <button
            onClick={() => handleResolve("keep-both")}
            disabled={resolving}
            style={{ ...actionButtonStyle, background: "var(--accent)", color: "#fff", border: "none" }}
          >
            Keep Both
          </button>
        </div>
      </div>
    </div>
  );
}

// --- Banner component for embedding above content area ---

interface SyncConflictBannerProps {
  conflictCount: number;
  onReview: () => void;
}

export function SyncConflictBanner({ conflictCount, onReview }: SyncConflictBannerProps) {
  if (conflictCount === 0) return null;

  return (
    <div style={bannerStyle}>
      <span style={{ fontSize: 12, color: "var(--yellow)" }}>
        {"\u26A0"} {conflictCount} sync conflict{conflictCount !== 1 ? "s" : ""}
      </span>
      <button onClick={onReview} style={bannerLinkStyle}>
        Review
      </button>
    </div>
  );
}

// --- Styles ---

const overlayStyle: CSSProperties = {
  position: "fixed",
  top: 0,
  left: 0,
  right: 0,
  bottom: 0,
  background: "rgba(0, 0, 0, 0.6)",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  zIndex: 500,
};

const dialogStyle: CSSProperties = {
  background: "var(--bg-2)",
  border: "1px solid var(--border)",
  borderRadius: "var(--radius)",
  width: "min(90vw, 800px)",
  maxHeight: "80vh",
  display: "flex",
  flexDirection: "column",
  boxShadow: "0 8px 32px rgba(0, 0, 0, 0.5)",
};

const headerStyle: CSSProperties = {
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  padding: "12px 16px",
  borderBottom: "1px solid var(--border)",
};

const closeButtonStyle: CSSProperties = {
  background: "transparent",
  border: "none",
  color: "var(--text-3)",
  cursor: "pointer",
  fontSize: 14,
  padding: "4px 6px",
  lineHeight: 1,
};

const navStyle: CSSProperties = {
  display: "flex",
  gap: 4,
  padding: "8px 16px",
  borderBottom: "1px solid var(--border)",
  flexWrap: "wrap",
};

const navButtonStyle: CSSProperties = {
  background: "transparent",
  border: "1px solid var(--border)",
  borderRadius: "var(--radius)",
  padding: "3px 8px",
  fontSize: 11,
  cursor: "pointer",
  fontFamily: "var(--font-mono)",
  transition: "background 0.1s",
};

const pathStyle: CSSProperties = {
  padding: "8px 16px",
  fontFamily: "var(--font-mono)",
  fontSize: 11,
  color: "var(--text-3)",
  borderBottom: "1px solid var(--border)",
};

const panesContainerStyle: CSSProperties = {
  display: "flex",
  flex: 1,
  overflow: "hidden",
  minHeight: 200,
  maxHeight: 400,
};

const paneStyle: CSSProperties = {
  flex: 1,
  display: "flex",
  flexDirection: "column",
  overflow: "hidden",
};

const paneLabelStyle: CSSProperties = {
  padding: "6px 12px",
  fontSize: 11,
  fontWeight: 600,
  color: "var(--text-2)",
  textTransform: "uppercase",
  letterSpacing: "0.04em",
  borderBottom: "1px solid var(--border)",
  background: "var(--bg-3)",
};

const paneContentStyle: CSSProperties = {
  flex: 1,
  overflow: "auto",
  padding: "8px 12px",
  margin: 0,
  fontFamily: "var(--font-mono)",
  fontSize: 11,
  color: "var(--text-2)",
  lineHeight: 1.6,
  whiteSpace: "pre-wrap",
  wordBreak: "break-word",
  background: "var(--bg)",
};

const actionsStyle: CSSProperties = {
  display: "flex",
  gap: 8,
  padding: "12px 16px",
  borderTop: "1px solid var(--border)",
  justifyContent: "flex-end",
};

const actionButtonStyle: CSSProperties = {
  background: "var(--bg-3)",
  border: "1px solid var(--border)",
  borderRadius: "var(--radius)",
  color: "var(--text-2)",
  padding: "7px 16px",
  fontSize: 12,
  fontWeight: 600,
  cursor: "pointer",
  fontFamily: "var(--font-ui)",
  transition: "background 0.15s",
};

const bannerStyle: CSSProperties = {
  display: "flex",
  alignItems: "center",
  gap: 8,
  padding: "6px 16px",
  background: "rgba(240, 192, 64, 0.08)",
  borderBottom: "1px solid var(--border)",
};

const bannerLinkStyle: CSSProperties = {
  background: "transparent",
  border: "none",
  color: "var(--accent)",
  fontSize: 12,
  fontWeight: 600,
  cursor: "pointer",
  fontFamily: "var(--font-ui)",
  padding: 0,
  textDecoration: "underline",
};
