package sync

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

const identityFile = "identity.key"

var wordList = []string{
	"alpha", "amber", "aqua", "arctic", "azure", "birch", "blaze", "bloom",
	"bolt", "breeze", "brook", "cedar", "cliff", "cloud", "cobalt", "coral",
	"crane", "creek", "crimson", "crystal", "dawn", "delta", "drift", "dusk",
	"eagle", "ember", "fern", "flame", "flint", "frost", "gale", "glacier",
	"grove", "harbor", "hawk", "heath", "heron", "hollow", "iron", "jade",
	"lake", "lark", "leaf", "light", "lotus", "maple", "marsh", "mist",
	"moon", "moss", "night", "oak", "olive", "onyx", "peak", "pine",
	"rain", "reef", "ridge", "river", "sage", "shadow", "shore", "silver",
}

// forgeDir returns the path to the ~/.agentdesk directory.
func dataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agentdesk")
}

// identityPath returns the full path to the identity key file.
func identityPath() string {
	return filepath.Join(dataDir(), identityFile)
}

// LoadOrCreateIdentity loads an existing Ed25519 private key from
// ~/.agentdesk/identity.key, or generates a new one and persists it there
// if no key file exists yet. The key file is written with 0600 permissions.
func LoadOrCreateIdentity() (crypto.PrivKey, error) {
	data, err := os.ReadFile(identityPath())
	if err == nil {
		return crypto.UnmarshalPrivateKey(data)
	}

	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	raw, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("marshal key: %w", err)
	}

	if err := os.MkdirAll(dataDir(), 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	if err := os.WriteFile(identityPath(), raw, 0600); err != nil {
		return nil, fmt.Errorf("write identity: %w", err)
	}

	return priv, nil
}

// PeerIDFromKey derives a libp2p peer.ID from the given private key.
func PeerIDFromKey(priv crypto.PrivKey) (peer.ID, error) {
	return peer.IDFromPrivateKey(priv)
}

// HumanReadableID returns a stable, human-readable "agentdesk-word-word-word"
// identifier derived deterministically from the given peer.ID.
//
// We hash the peer ID with SHA-256 and slice the digest into three 32-bit
// chunks. Each chunk picks a word from the list. Hashing is necessary because
// libp2p peer IDs share a fixed multibase/multicodec prefix; without hashing,
// the high-order bytes are constant across identities and the human ID
// collapses to a small handful of word combinations.
func HumanReadableID(pid peer.ID) string {
	digest := sha256.Sum256([]byte(pid))
	words := make([]string, 3)
	for i := 0; i < 3; i++ {
		// Read 4 bytes per word; SHA-256 gives us 32, plenty.
		chunk := binary.BigEndian.Uint32(digest[i*4 : i*4+4])
		words[i] = wordList[chunk%uint32(len(wordList))]
	}
	return "agentdesk-" + strings.Join(words, "-")
}
