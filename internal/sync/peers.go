package sync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// PeerInfo holds the metadata for a paired remote peer.
type PeerInfo struct {
	ID        string `json:"id"`
	HumanID   string `json:"humanId"`
	Name      string `json:"name"`
	PairedAt  int64  `json:"pairedAt"`
	LastSeen  int64  `json:"lastSeen"`
	Connected bool   `json:"connected"`
}

// PeerStore is the in-memory representation of all paired peers, backed by
// a JSON file at ~/.agentdesk/peers.json.
type PeerStore struct {
	Peers []PeerInfo `json:"peers"`
}

// peersPath returns the absolute path to the peers persistence file.
func peersPath() string {
	return filepath.Join(dataDir(), "peers.json")
}

// LoadPeers reads the peer store from disk. If the file does not exist or
// cannot be decoded, an empty PeerStore is returned so callers can continue
// safely without error handling at call sites.
func LoadPeers() PeerStore {
	data, err := os.ReadFile(peersPath())
	if err != nil {
		return PeerStore{}
	}
	var store PeerStore
	if err := json.Unmarshal(data, &store); err != nil {
		return PeerStore{}
	}
	return store
}

// SavePeers writes the peer store to ~/.agentdesk/peers.json, creating the
// directory if it does not already exist.
func SavePeers(store PeerStore) error {
	if err := os.MkdirAll(dataDir(), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(peersPath(), data, 0644)
}

// AddPeer adds a new peer to the store, or updates the name and LastSeen
// timestamp if the peer is already paired.
func (ps *PeerStore) AddPeer(pid peer.ID, name string) {
	for i, p := range ps.Peers {
		if p.ID == pid.String() {
			ps.Peers[i].Name = name
			ps.Peers[i].LastSeen = time.Now().Unix()
			return
		}
	}
	ps.Peers = append(ps.Peers, PeerInfo{
		ID:       pid.String(),
		HumanID:  HumanReadableID(pid),
		Name:     name,
		PairedAt: time.Now().Unix(),
		LastSeen: time.Now().Unix(),
	})
}

// RemovePeer removes the peer with the given string ID from the store.
// It is a no-op if the peer is not found.
func (ps *PeerStore) RemovePeer(peerID string) {
	filtered := ps.Peers[:0]
	for _, p := range ps.Peers {
		if p.ID != peerID {
			filtered = append(filtered, p)
		}
	}
	ps.Peers = filtered
}

// IsPaired reports whether the given peer ID is in the store.
func (ps *PeerStore) IsPaired(pid peer.ID) bool {
	for _, p := range ps.Peers {
		if p.ID == pid.String() {
			return true
		}
	}
	return false
}

// UpdateLastSeen refreshes the LastSeen timestamp for the given peer.
// It is a no-op if the peer is not found.
func (ps *PeerStore) UpdateLastSeen(pid peer.ID) {
	for i, p := range ps.Peers {
		if p.ID == pid.String() {
			ps.Peers[i].LastSeen = time.Now().Unix()
			return
		}
	}
}
