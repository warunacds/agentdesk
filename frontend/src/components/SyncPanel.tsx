import { useState, useEffect, useCallback } from "react";
import { SyncStatus, SyncPeer } from "../types";

// These imports will resolve after `wails build` generates the bindings.
// Until then, TypeScript will report import errors — that is expected.
import {
  GetDeviceID,
  GetPeers,
  AddPeerByID,
  RemovePeerByID,
  GetFullPeerID,
} from "../../wailsjs/go/main/App";

interface SyncPanelProps {
  syncStatus: SyncStatus | null;
  onRefresh: () => void;
}

export function SyncPanel({ syncStatus, onRefresh }: SyncPanelProps) {
  const [humanId, setHumanId] = useState<string>("");
  const [fullPeerId, setFullPeerId] = useState<string>("");
  const [peers, setPeers] = useState<SyncPeer[]>([]);
  const [peerIdInput, setPeerIdInput] = useState("");
  const [peerNameInput, setPeerNameInput] = useState("");
  const [pairing, setPairing] = useState(false);
  const [copiedField, setCopiedField] = useState<string | null>(null);

  const loadPeers = useCallback(async () => {
    try {
      const p = await GetPeers();
      setPeers((p ?? []) as unknown as SyncPeer[]);
    } catch {
      // Sync methods may not exist yet
    }
  }, []);

  const loadIdentity = useCallback(async () => {
    try {
      const hid = await GetDeviceID();
      setHumanId(hid ?? "");
    } catch {
      // Sync methods may not exist yet
    }
    try {
      const fid = await GetFullPeerID();
      setFullPeerId(fid ?? "");
    } catch {
      // Sync methods may not exist yet
    }
  }, []);

  useEffect(() => {
    loadIdentity();
    loadPeers();
  }, [loadIdentity, loadPeers]);

  // Re-load peers whenever syncStatus changes (e.g., discovery event)
  useEffect(() => {
    loadPeers();
  }, [syncStatus, loadPeers]);

  const handleCopy = async (text: string, field: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedField(field);
      setTimeout(() => setCopiedField(null), 1500);
    } catch {
      // Clipboard not available
    }
  };

  const handlePair = async () => {
    const id = peerIdInput.trim();
    const name = peerNameInput.trim();
    if (!id) return;
    setPairing(true);
    try {
      await AddPeerByID(id, name || "Unnamed Device");
      setPeerIdInput("");
      setPeerNameInput("");
      await loadPeers();
      onRefresh();
    } catch {
      // Pairing may fail
    } finally {
      setPairing(false);
    }
  };

  const handleRemovePeer = async (peerId: string) => {
    try {
      await RemovePeerByID(peerId);
      await loadPeers();
      onRefresh();
    } catch {
      // Removal may fail
    }
  };

  const truncatePeerId = (id: string, max: number = 32): string => {
    if (!id || id.length <= max) return id;
    return id.slice(0, max) + "\u2026";
  };

  return (
    <div>
      {/* Your Device ID */}
      <div style={{ marginBottom: 16 }}>
        <div style={labelStyle}>Your Device ID</div>
        <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
          <div
            style={{
              flex: 1,
              background: "var(--bg-3)",
              borderRadius: "var(--radius)",
              padding: "7px 10px",
              fontFamily: "var(--font-mono)",
              fontSize: 12,
              color: "var(--text)",
              border: "1px solid var(--border)",
              overflow: "hidden",
              textOverflow: "ellipsis",
              whiteSpace: "nowrap",
            }}
          >
            {humanId || "\u2014"}
          </div>
          <button
            onClick={() => handleCopy(humanId, "humanId")}
            style={copyButtonStyle}
            title="Copy Device ID"
          >
            {copiedField === "humanId" ? "\u2713" : "\u2398"}
          </button>
        </div>
        <div
          onClick={() => handleCopy(fullPeerId, "fullPeerId")}
          style={{
            fontSize: 10,
            color: copiedField === "fullPeerId" ? "var(--green)" : "var(--text-3)",
            marginTop: 4,
            fontFamily: "var(--font-mono)",
            cursor: fullPeerId ? "pointer" : "default",
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
            transition: "color 0.15s",
          }}
          title={fullPeerId || "Full peer ID"}
        >
          {copiedField === "fullPeerId"
            ? "Copied!"
            : fullPeerId
            ? truncatePeerId(fullPeerId, 48) + " (click to copy)"
            : "Full peer ID"}
        </div>
      </div>

      {/* Paired Devices */}
      <div style={{ marginBottom: 16 }}>
        <div style={labelStyle}>Paired Devices</div>
        {peers.length === 0 ? (
          <div
            style={{
              fontSize: 12,
              color: "var(--text-3)",
              padding: "10px",
              background: "var(--bg-3)",
              borderRadius: "var(--radius)",
              border: "1px solid var(--border)",
              textAlign: "center",
            }}
          >
            No paired devices
          </div>
        ) : (
          <div
            style={{
              background: "var(--bg-3)",
              borderRadius: "var(--radius)",
              border: "1px solid var(--border)",
              overflow: "hidden",
            }}
          >
            {peers.map((peer, idx) => (
              <div
                key={peer.id}
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: 8,
                  padding: "7px 10px",
                  borderTop: idx > 0 ? "1px solid var(--border)" : "none",
                }}
              >
                {/* Connection dot */}
                <span
                  style={{
                    width: 8,
                    height: 8,
                    borderRadius: "50%",
                    background: peer.connected ? "var(--green)" : "var(--text-3)",
                    flexShrink: 0,
                  }}
                  title={peer.connected ? "Connected" : "Disconnected"}
                />
                {/* Name */}
                <span
                  style={{
                    fontSize: 12,
                    color: "var(--text)",
                    fontFamily: "var(--font-ui)",
                    fontWeight: 500,
                    flexShrink: 0,
                  }}
                >
                  {peer.name}
                </span>
                {/* Human ID */}
                <span
                  style={{
                    flex: 1,
                    fontSize: 11,
                    color: "var(--text-3)",
                    fontFamily: "var(--font-mono)",
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                  }}
                  title={peer.humanId}
                >
                  {peer.humanId}
                </span>
                {/* Remove button */}
                <button
                  onClick={() => handleRemovePeer(peer.id)}
                  style={{
                    background: "transparent",
                    border: "none",
                    color: "var(--text-3)",
                    cursor: "pointer",
                    fontSize: 13,
                    padding: "0 4px",
                    lineHeight: 1,
                    flexShrink: 0,
                    transition: "color 0.15s",
                  }}
                  title="Remove peer"
                  onMouseEnter={(e) => {
                    e.currentTarget.style.color = "var(--red)";
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.color = "var(--text-3)";
                  }}
                >
                  {"\u2715"}
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Add Device */}
      <div style={{ marginBottom: 16 }}>
        <div style={labelStyle}>Add Device</div>
        <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
          <input
            type="text"
            placeholder="Device name..."
            value={peerNameInput}
            onChange={(e) => setPeerNameInput(e.target.value)}
            style={inputStyle}
          />
          <div style={{ display: "flex", gap: 6 }}>
            <input
              type="text"
              placeholder="Paste peer ID here..."
              value={peerIdInput}
              onChange={(e) => setPeerIdInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") handlePair();
              }}
              style={{ ...inputStyle, flex: 1 }}
            />
            <button
              onClick={handlePair}
              disabled={pairing || !peerIdInput.trim()}
              style={{
                background: peerIdInput.trim() ? "var(--accent)" : "var(--bg-3)",
                border: "none",
                borderRadius: "var(--radius)",
                color: peerIdInput.trim() ? "#fff" : "var(--text-3)",
                padding: "6px 14px",
                fontSize: 11,
                fontWeight: 600,
                cursor: peerIdInput.trim() ? "pointer" : "default",
                fontFamily: "var(--font-ui)",
                flexShrink: 0,
                transition: "background 0.15s, color 0.15s",
              }}
            >
              {pairing ? "\u2026" : "Pair"}
            </button>
          </div>
        </div>
      </div>

      {/* Status line */}
      {syncStatus && syncStatus.running && (
        <div
          style={{
            fontSize: 11,
            color: "var(--text-3)",
            fontFamily: "var(--font-ui)",
            padding: "6px 0 0",
            display: "flex",
            alignItems: "center",
            gap: 6,
          }}
        >
          <span
            style={{
              width: 6,
              height: 6,
              borderRadius: "50%",
              background: syncStatus.connectedPeers > 0 ? "var(--green)" : "var(--text-3)",
              flexShrink: 0,
            }}
          />
          <span>
            Sync: {syncStatus.trackedFiles} file{syncStatus.trackedFiles !== 1 ? "s" : ""} tracked
            {" \u00B7 "}
            {syncStatus.connectedPeers} peer{syncStatus.connectedPeers !== 1 ? "s" : ""} connected
          </span>
        </div>
      )}
    </div>
  );
}

const labelStyle: React.CSSProperties = {
  fontSize: 13,
  color: "var(--text)",
  fontFamily: "var(--font-ui)",
  fontWeight: 500,
  marginBottom: 6,
};

const copyButtonStyle: React.CSSProperties = {
  background: "transparent",
  border: "1px solid var(--border)",
  borderRadius: "var(--radius)",
  color: "var(--text-3)",
  padding: "5px 8px",
  fontSize: 12,
  cursor: "pointer",
  fontFamily: "var(--font-mono)",
  lineHeight: 1,
  flexShrink: 0,
};

const inputStyle: React.CSSProperties = {
  background: "var(--bg-3)",
  border: "1px solid var(--border)",
  borderRadius: "var(--radius)",
  color: "var(--text)",
  padding: "6px 10px",
  fontSize: 12,
  fontFamily: "var(--font-mono)",
  outline: "none",
};
