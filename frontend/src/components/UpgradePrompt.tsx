export function UpgradePrompt() {
  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        padding: "32px 20px",
        textAlign: "center",
      }}
    >
      {/* Hex icon */}
      <div style={{ fontSize: 36, color: "var(--accent)", marginBottom: 16, opacity: 0.8 }}>
        {"\u2b21"}
      </div>

      {/* Heading */}
      <div style={{ fontSize: 16, fontWeight: 600, color: "var(--text)", fontFamily: "var(--font-ui)", marginBottom: 8 }}>
        P2P Sync
      </div>

      {/* Coming Soon badge */}
      <div style={{
        fontSize: 10,
        fontWeight: 700,
        fontFamily: "var(--font-mono)",
        color: "var(--yellow)",
        background: "rgba(240,192,64,0.15)",
        padding: "3px 10px",
        borderRadius: 4,
        letterSpacing: "0.05em",
        marginBottom: 16,
      }}>
        COMING SOON
      </div>

      {/* Description */}
      <div style={{ fontSize: 12, color: "var(--text-3)", lineHeight: 1.6, maxWidth: 320, marginBottom: 20 }}>
        Sync your skill files across devices with encrypted peer-to-peer connections. No cloud servers, no accounts.
      </div>

      {/* Feature list */}
      <div style={{ display: "flex", flexDirection: "column", gap: 8, alignItems: "flex-start", width: "100%", maxWidth: 280 }}>
        {[
          "Automatic file sync across devices",
          "End-to-end encrypted",
          "Works on LAN + internet",
          "Free and open source",
        ].map((feature) => (
          <div key={feature} style={{ display: "flex", alignItems: "center", gap: 8, fontSize: 12, color: "var(--text-2)", fontFamily: "var(--font-ui)" }}>
            <span style={{ color: "var(--green)", fontSize: 13, flexShrink: 0, fontWeight: 600 }}>{"\u2713"}</span>
            {feature}
          </div>
        ))}
      </div>
    </div>
  );
}
