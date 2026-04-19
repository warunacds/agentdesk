import React from "react";

interface Props {
  children: React.ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends React.Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    console.error("Agent Desk error boundary caught:", error, info.componentStack);
  }

  render() {
    if (this.state.hasError) {
      return (
        <div
          style={{
            flex: 1,
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            justifyContent: "center",
            gap: 16,
            padding: 40,
            background: "var(--bg)",
            color: "var(--text)",
          }}
        >
          <div style={{ fontSize: 32, opacity: 0.4 }}>{"\u26A0"}</div>
          <div style={{ fontSize: 15, fontWeight: 600, color: "var(--text)" }}>
            Something went wrong
          </div>
          <div
            style={{
              fontSize: 12,
              color: "var(--text-3)",
              fontFamily: "var(--font-mono)",
              maxWidth: 400,
              textAlign: "center",
              lineHeight: 1.6,
              wordBreak: "break-word",
            }}
          >
            {this.state.error?.message || "Unknown error"}
          </div>
          <button
            onClick={() => this.setState({ hasError: false, error: null })}
            style={{
              marginTop: 8,
              background: "var(--accent)",
              border: "none",
              borderRadius: "var(--radius)",
              color: "#fff",
              padding: "8px 20px",
              fontSize: 13,
              fontWeight: 600,
              cursor: "pointer",
              fontFamily: "var(--font-ui)",
            }}
          >
            Reload
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}
