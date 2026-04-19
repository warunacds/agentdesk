// Package sync provides peer-to-peer synchronisation primitives built on libp2p.
// It defines the message wire format, protocol helpers, and the Engine that
// orchestrates file synchronisation between paired Forge peers.
package sync

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/libp2p/go-libp2p"
	_ "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	_ "github.com/multiformats/go-multiaddr"
)

// ProtocolID is the libp2p protocol identifier for the Forge sync stream.
const ProtocolID = "/agentdesk/sync/1.0.0"

// ---------------------------------------------------------------------------
// Message types
// ---------------------------------------------------------------------------

// MessageType identifies the kind of sync message on the wire.
type MessageType string

const (
	// MsgFileAnnounce announces a local file change to peers.
	MsgFileAnnounce MessageType = "file-announce"
	// MsgFileRequest requests the content of a file from a peer.
	MsgFileRequest MessageType = "file-request"
	// MsgFileData carries the base64-encoded content of a file.
	MsgFileData MessageType = "file-data"
	// MsgFileDelete announces a local file deletion to peers.
	MsgFileDelete MessageType = "file-delete"
	// MsgSyncRequest requests the full sync state from a peer.
	MsgSyncRequest MessageType = "sync-request"
	// MsgSyncState carries the full sync state in response to MsgSyncRequest.
	MsgSyncState MessageType = "sync-state"
)

// SyncMessage is the envelope for every message sent over the sync protocol.
type SyncMessage struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// FileAnnouncePayload is the payload for MsgFileAnnounce.
type FileAnnouncePayload struct {
	Path     string            `json:"path"`
	Hash     string            `json:"hash"`
	Versions map[string]uint64 `json:"versions"`
	Modified int64             `json:"modified"`
	Size     int64             `json:"size"`
}

// FileRequestPayload is the payload for MsgFileRequest.
type FileRequestPayload struct {
	Path string `json:"path"`
}

// FileDataPayload is the payload for MsgFileData.
type FileDataPayload struct {
	Path     string            `json:"path"`
	Content  string            `json:"content"` // base64 encoded
	Hash     string            `json:"hash"`
	Versions map[string]uint64 `json:"versions"`
	Modified int64             `json:"modified"`
	Size     int64             `json:"size"`
}

// FileDeletePayload is the payload for MsgFileDelete.
type FileDeletePayload struct {
	Path      string            `json:"path"`
	Versions  map[string]uint64 `json:"versions"`
	DeletedAt int64             `json:"deletedAt"`
}

// SyncStatePayload is the payload for MsgSyncState.
type SyncStatePayload struct {
	Files map[string]*FileVersion `json:"files"`
}

// EncodeMessage serialises a typed payload into a SyncMessage wire format.
func EncodeMessage(msgType MessageType, payload interface{}) ([]byte, error) {
	p, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return json.Marshal(SyncMessage{Type: msgType, Payload: p})
}

// DecodeMessage deserialises a SyncMessage from its wire format.
func DecodeMessage(data []byte) (SyncMessage, error) {
	var msg SyncMessage
	err := json.Unmarshal(data, &msg)
	return msg, err
}

// unmarshalPayload is a generic helper that extracts a typed payload from a
// SyncMessage.
func unmarshalPayload[T any](msg SyncMessage) (T, error) {
	var t T
	err := json.Unmarshal(msg.Payload, &t)
	return t, err
}

// ---------------------------------------------------------------------------
// NodeHost — interface that Engine uses to talk to the network layer
// ---------------------------------------------------------------------------

// NodeHost abstracts the network operations that the Engine requires from the
// Node. Defining this as an interface keeps engine.go compilable before
// node.go exists and makes it straightforward to test without a live libp2p
// host.
type NodeHost interface {
	// BroadcastToPairedPeers sends raw message bytes to every connected
	// paired peer.
	BroadcastToPairedPeers(data []byte)
	// SendToPeer sends raw message bytes to a specific peer.
	SendToPeer(pid peer.ID, data []byte) error
	// EmitEvent notifies the frontend of a sync-related event.
	EmitEvent(event string, data map[string]string)
	// PeerID returns the local peer ID.
	PeerID() peer.ID
	// HumanID returns the human-readable identifier for the local peer.
	HumanID() string
	// Connectedness reports the connection state for a given peer.
	Connectedness(pid peer.ID) network.Connectedness
}

// ---------------------------------------------------------------------------
// Protocol helpers
// ---------------------------------------------------------------------------

// AddPeerByString is a convenience function for callers that have the peer ID
// as a string (e.g. the agentdesk-core CLI). It decodes the string, adds the peer
// to the engine, and initiates a connection via the node.
func AddPeerByString(engine *Engine, node *Node, peerIDStr, name string) error {
	pid, err := peer.Decode(peerIDStr)
	if err != nil {
		return fmt.Errorf("invalid peer ID: %w", err)
	}
	if err := engine.AddPeer(pid, name); err != nil {
		return err
	}
	go node.ConnectToPeer(pid)
	return nil
}

// writeResponse writes a message back on an existing stream, appending a
// newline delimiter.
func writeResponse(s network.Stream, data []byte) error {
	data = append(data, '\n')
	_, err := s.Write(data)
	return err
}

// nowUnix returns the current time as a Unix timestamp.
func nowUnix() int64 { return time.Now().Unix() }

// readFileBase64 reads the file at path and returns its content as a
// base64-encoded string.
func readFileBase64(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// writeFileBase64 decodes a base64 string and writes the resulting bytes
// to path, creating parent directories as needed.
func writeFileBase64(path string, b64content string) error {
	data, err := base64.StdEncoding.DecodeString(b64content)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// archiveConflict saves a timestamped conflict copy of the file at absPath
// next to the original. If the file cannot be read an error is returned.
func archiveConflict(absPath string) error {
	ext := filepath.Ext(absPath)
	base := strings.TrimSuffix(absPath, ext)
	ts := time.Now().Format("20060102-150405")
	conflictPath := fmt.Sprintf("%s.conflict-%s%s", base, ts, ext)

	content, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}
	return os.WriteFile(conflictPath, content, 0644)
}

// trashFile moves the file at absPath into ~/.agentdesk/trash/ with a timestamped
// name instead of permanently deleting it.
func trashFile(absPath string) error {
	trashDir := filepath.Join(dataDir(), "trash")
	if err := os.MkdirAll(trashDir, 0755); err != nil {
		return err
	}
	filename := filepath.Base(absPath)
	ts := time.Now().Format("20060102-150405")
	dest := filepath.Join(trashDir, ts+"-"+filename)
	return os.Rename(absPath, dest)
}
