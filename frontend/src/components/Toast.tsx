import { useEffect, useState, CSSProperties } from "react";

interface ToastData {
  message: string;
  undoAction: () => void;
}

interface ToastProps {
  toast: ToastData | null;
  onDismiss: () => void;
}

export function Toast({ toast, onDismiss }: ToastProps) {
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    if (toast) {
      // Trigger fade-in on next frame
      requestAnimationFrame(() => setVisible(true));

      const timer = setTimeout(() => {
        setVisible(false);
        // Wait for fade-out transition before dismissing
        setTimeout(onDismiss, 200);
      }, 5000);

      return () => clearTimeout(timer);
    } else {
      setVisible(false);
    }
  }, [toast, onDismiss]);

  if (!toast) return null;

  const handleUndo = () => {
    toast.undoAction();
    setVisible(false);
    setTimeout(onDismiss, 200);
  };

  const containerStyle: CSSProperties = {
    position: "fixed",
    bottom: 20,
    left: "50%",
    transform: "translateX(-50%)",
    background: "var(--bg-3)",
    border: "1px solid var(--border)",
    borderRadius: "var(--radius)",
    padding: "10px 16px",
    display: "flex",
    alignItems: "center",
    gap: 16,
    boxShadow: "0 4px 16px rgba(0, 0, 0, 0.3)",
    zIndex: 1000,
    opacity: visible ? 1 : 0,
    transition: "opacity 0.2s ease",
    pointerEvents: visible ? "auto" : "none",
  };

  const messageStyle: CSSProperties = {
    color: "var(--text-2)",
    fontSize: 13,
    fontFamily: "var(--font-ui)",
    whiteSpace: "nowrap",
  };

  const undoButtonStyle: CSSProperties = {
    background: "transparent",
    border: "none",
    color: "var(--accent)",
    fontSize: 13,
    fontWeight: 700,
    cursor: "pointer",
    fontFamily: "var(--font-ui)",
    padding: "2px 4px",
    whiteSpace: "nowrap",
  };

  return (
    <div style={containerStyle}>
      <span style={messageStyle}>{toast.message}</span>
      <button onClick={handleUndo} style={undoButtonStyle}>
        Undo
      </button>
    </div>
  );
}
