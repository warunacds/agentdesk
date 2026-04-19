package sync

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	gosync "sync"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Engine orchestrates file synchronisation between paired Forge peers. It
// owns the local SyncState, PeerStore, and SyncManifest, compares version
// vectors to determine which side is newer, resolves conflicts, and drives
// file transfers through the NodeHost interface.
type Engine struct {
	node             NodeHost
	state            *SyncState
	peers            PeerStore
	manifest         *SyncManifest
	pendingConflicts map[string]pendingConflict // keyed by relPath
	mu               gosync.Mutex
	stopCh           chan struct{}
}

// pendingConflict stores information needed to construct a SyncConflict once
// the remote file data arrives asynchronously via handleFileData.
type pendingConflict struct {
	localContent string
	localHash    string
	remotePeer   string
}

// NewEngine creates an Engine wired to the given NodeHost. It loads the
// persisted SyncState, PeerStore, and SyncManifest from disk, setting the
// local peer ID on the state so that vector-clock increments are attributed
// correctly.
func NewEngine(node NodeHost) *Engine {
	state := LoadSyncState()
	state.LocalID = node.PeerID().String()
	peers := LoadPeers()
	manifest := LoadManifest()

	return &Engine{
		node:             node,
		state:            state,
		peers:            peers,
		manifest:         manifest,
		pendingConflicts: make(map[string]pendingConflict),
		stopCh:           make(chan struct{}),
	}
}

// Start kicks off the initial reconciliation in the background. It scans
// local files and updates the sync state for anything that changed while
// Forge was not running.
func (e *Engine) Start() {
	go e.reconcile()
}

// Stop persists the current SyncState, PeerStore, and SyncManifest to disk
// and signals the engine to shut down.
func (e *Engine) Stop() {
	close(e.stopCh)
	e.mu.Lock()
	defer e.mu.Unlock()
	_ = e.state.Save()
	_ = SavePeers(e.peers)
	_ = SaveManifest(e.manifest)
}

// IsPairedPeer reports whether pid is in the paired peer list.
func (e *Engine) IsPairedPeer(pid peer.ID) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.peers.IsPaired(pid)
}

// GetPairedPeers returns a snapshot of all paired peers. The returned
// slice is a copy so callers cannot mutate internal state.
func (e *Engine) GetPairedPeers() []PeerInfo {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]PeerInfo{}, e.peers.Peers...)
}

// AddPeer pairs with a new peer identified by pid and persists the change
// to disk.
func (e *Engine) AddPeer(pid peer.ID, name string) error {
	e.mu.Lock()
	e.peers.AddPeer(pid, name)
	err := SavePeers(e.peers)
	e.mu.Unlock()
	return err
}

// RemovePeer unpairs the peer with the given string ID and persists the
// change to disk.
func (e *Engine) RemovePeer(peerID string) error {
	e.mu.Lock()
	e.peers.RemovePeer(peerID)
	err := SavePeers(e.peers)
	e.mu.Unlock()
	return err
}

// OnLocalFileChange is called by the fsnotify watcher whenever a file
// inside a watched directory is created or modified. It updates the local
// sync state and broadcasts an announcement to all paired peers. If the
// file was deleted it broadcasts a deletion message instead.
func (e *Engine) OnLocalFileChange(absPath string) {
	if !IsSyncable(absPath) {
		return
	}
	relPath := toRelPath(absPath)
	if SkipSyncFile(relPath) {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// In selective mode, only sync files that are opted-in via the manifest.
	if !e.manifest.IsPathSynced(relPath) {
		return
	}

	hash, err := HashFile(absPath)
	if err != nil {
		// File might have been deleted
		if os.IsNotExist(err) {
			e.handleLocalDeletion(relPath)
		}
		return
	}

	// Check if the hash actually changed
	if fv, ok := e.state.Files[relPath]; ok && fv.Hash == hash {
		return // no real change
	}

	fi, err := os.Stat(absPath)
	if err != nil {
		return
	}

	e.state.RecordLocalChange(relPath, hash, fi.ModTime().Unix(), fi.Size())
	_ = e.state.Save()

	// Broadcast announcement to peers
	fv := e.state.Files[relPath]
	msg, err := EncodeMessage(MsgFileAnnounce, FileAnnouncePayload{
		Path:     relPath,
		Hash:     hash,
		Versions: fv.Versions,
		Modified: fv.Modified,
		Size:     fv.Size,
	})
	if err != nil {
		return
	}
	e.node.BroadcastToPairedPeers(msg)
}

// handleLocalDeletion records a deletion in the sync state and broadcasts
// the event to peers. The caller must hold e.mu.
func (e *Engine) handleLocalDeletion(relPath string) {
	now := nowUnix()
	e.state.RecordDeletion(relPath, now)
	_ = e.state.Save()

	fv := e.state.Files[relPath]
	msg, _ := EncodeMessage(MsgFileDelete, FileDeletePayload{
		Path:      relPath,
		Versions:  fv.Versions,
		DeletedAt: now,
	})
	e.node.BroadcastToPairedPeers(msg)
}

// HandleIncoming satisfies the SyncEngine interface. It decodes the raw
// inbound bytes into a SyncMessage and delegates to HandleMessage for
// type-based dispatch. If the bytes cannot be decoded the message is
// silently dropped.
func (e *Engine) HandleIncoming(from peer.ID, data []byte, s network.Stream) {
	msg, err := DecodeMessage(data)
	if err != nil {
		log.Printf("sync: failed to decode message from %s: %v", from, err)
		return
	}
	e.HandleMessage(from, msg, s)
}

// HandleMessage processes an incoming sync message from a remote peer.
// If the peer is not yet paired, it is auto-paired (mutual trust: if they
// know our peer ID and send us a valid message, we trust them back).
func (e *Engine) HandleMessage(from peer.ID, msg SyncMessage, s network.Stream) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Auto-pair: if someone sends us a valid sync message, pair them back.
	// This makes pairing bidirectional without requiring both sides to
	// manually enter each other's IDs.
	if !e.peers.IsPaired(from) {
		log.Printf("sync: auto-pairing peer %s (they initiated sync)", from)
		e.peers.AddPeer(from, "")
		_ = SavePeers(e.peers)
		e.node.EmitEvent("peer-auto-paired", map[string]string{
			"peerId": from.String(),
		})
	}

	switch msg.Type {
	case MsgFileAnnounce:
		e.handleFileAnnounce(from, msg, s)
	case MsgFileRequest:
		e.handleFileRequest(msg, s)
	case MsgFileData:
		e.handleFileData(from, msg)
	case MsgFileDelete:
		e.handleFileDelete(from, msg)
	case MsgSyncRequest:
		e.handleSyncRequest(s)
	case MsgSyncState:
		e.handleSyncState(from, msg)
	}
}

// handleFileAnnounce compares the announced file's version vector against
// the local state and either requests the file (if remote is newer),
// records a conflict for user resolution (if concurrent), or does nothing
// (if local is newer or equal). Files not in the sync manifest are ignored.
func (e *Engine) handleFileAnnounce(from peer.ID, msg SyncMessage, s network.Stream) {
	payload, err := unmarshalPayload[FileAnnouncePayload](msg)
	if err != nil {
		return
	}

	// In selective mode, only accept files that are opted-in via the manifest.
	if !e.manifest.IsPathSynced(payload.Path) {
		return
	}

	localFV := e.state.Files[payload.Path]
	if localFV == nil {
		// We don't have this file -- request it
		req, _ := EncodeMessage(MsgFileRequest, FileRequestPayload{Path: payload.Path})
		_ = writeResponse(s, req)
		return
	}

	order := CompareVersions(localFV.Versions, payload.Versions)
	switch order {
	case RemoteNewer:
		req, _ := EncodeMessage(MsgFileRequest, FileRequestPayload{Path: payload.Path})
		_ = writeResponse(s, req)
	case Concurrent:
		// Store a pending conflict so we can capture the remote content
		// when handleFileData delivers it. Read local content now while
		// we still have the original version on disk.
		absPath := resolveAbsPath(payload.Path)
		localBytes, _ := os.ReadFile(absPath)
		e.pendingConflicts[payload.Path] = pendingConflict{
			localContent: string(localBytes),
			localHash:    localFV.Hash,
			remotePeer:   from.String(),
		}
		e.node.EmitEvent("sync-conflict", map[string]string{
			"path":   payload.Path,
			"peerId": from.String(),
		})
		// Request the remote version so we can capture its content
		req, _ := EncodeMessage(MsgFileRequest, FileRequestPayload{Path: payload.Path})
		_ = writeResponse(s, req)
	case LocalNewer, Equal:
		// We are up to date; nothing to do.
	}
}

// handleFileRequest reads the requested file from disk and sends its
// base64-encoded content back over the existing stream.
func (e *Engine) handleFileRequest(msg SyncMessage, s network.Stream) {
	payload, err := unmarshalPayload[FileRequestPayload](msg)
	if err != nil {
		return
	}

	absPath := resolveAbsPath(payload.Path)
	b64, err := readFileBase64(absPath)
	if err != nil {
		return
	}

	fv := e.state.Files[payload.Path]
	if fv == nil {
		return
	}

	resp, _ := EncodeMessage(MsgFileData, FileDataPayload{
		Path:     payload.Path,
		Content:  b64,
		Hash:     fv.Hash,
		Versions: fv.Versions,
		Modified: fv.Modified,
		Size:     fv.Size,
	})
	_ = writeResponse(s, resp)
}

// handleFileData writes the received file to disk, merges version vectors
// with the local state, and emits a sync event for the UI. If the file has
// a pending conflict (from a Concurrent version comparison), the conflict
// is recorded for user resolution instead of overwriting the local copy.
func (e *Engine) handleFileData(from peer.ID, msg SyncMessage) {
	payload, err := unmarshalPayload[FileDataPayload](msg)
	if err != nil {
		return
	}

	// Check if this file data corresponds to a pending conflict.
	if pc, ok := e.pendingConflicts[payload.Path]; ok {
		delete(e.pendingConflicts, payload.Path)

		// Decode remote content from base64 so we can store it.
		remoteBytes, decErr := base64.StdEncoding.DecodeString(payload.Content)
		remoteContent := ""
		if decErr == nil {
			remoteContent = string(remoteBytes)
		}

		conflict := SyncConflict{
			Path:          payload.Path,
			LocalHash:     pc.localHash,
			RemoteHash:    payload.Hash,
			RemotePeer:    pc.remotePeer,
			DetectedAt:    nowUnix(),
			LocalContent:  pc.localContent,
			RemoteContent: remoteContent,
		}
		e.state.Conflicts = append(e.state.Conflicts, conflict)
		_ = e.state.Save()

		e.node.EmitEvent("sync-conflict-stored", map[string]string{
			"path":   payload.Path,
			"peerId": from.String(),
		})
		return
	}

	absPath := resolveAbsPath(payload.Path)
	if err := writeFileBase64(absPath, payload.Content); err != nil {
		log.Printf("sync: failed to write %s: %v", payload.Path, err)
		return
	}

	// Merge version vectors and record the remote change
	localVersions := map[string]uint64{}
	if fv, ok := e.state.Files[payload.Path]; ok {
		localVersions = fv.Versions
	}
	merged := MergeVersions(localVersions, payload.Versions)

	e.state.RecordRemoteChange(payload.Path, FileVersion{
		Hash:     payload.Hash,
		Modified: payload.Modified,
		Size:     payload.Size,
		Versions: merged,
	})
	_ = e.state.Save()

	e.node.EmitEvent("file-synced", map[string]string{
		"path":   payload.Path,
		"peerId": from.String(),
	})
}

// handleFileDelete moves the local copy of the file to the trash
// directory and records the deletion in the sync state.
func (e *Engine) handleFileDelete(from peer.ID, msg SyncMessage) {
	payload, err := unmarshalPayload[FileDeletePayload](msg)
	if err != nil {
		return
	}

	absPath := resolveAbsPath(payload.Path)
	if _, err := os.Stat(absPath); err == nil {
		_ = trashFile(absPath)
	}

	e.state.RecordRemoteChange(payload.Path, FileVersion{
		Versions:  payload.Versions,
		Deleted:   true,
		DeletedAt: payload.DeletedAt,
	})
	_ = e.state.Save()

	e.node.EmitEvent("file-deleted-remote", map[string]string{
		"path":   payload.Path,
		"peerId": from.String(),
	})
}

// handleSyncRequest responds with the full local sync state so the
// requesting peer can compare and request any files it is missing.
func (e *Engine) handleSyncRequest(s network.Stream) {
	resp, _ := EncodeMessage(MsgSyncState, SyncStatePayload{
		Files: e.state.Files,
	})
	_ = writeResponse(s, resp)
}

// handleSyncState compares every file in the remote state against the
// local state and requests any files where the remote version is newer
// or the file is entirely missing locally.
func (e *Engine) handleSyncState(from peer.ID, msg SyncMessage) {
	payload, err := unmarshalPayload[SyncStatePayload](msg)
	if err != nil {
		return
	}

	for relPath, remoteFV := range payload.Files {
		if SkipSyncFile(relPath) {
			continue
		}
		// In selective mode, only accept files that are opted-in.
		if !e.manifest.IsPathSynced(relPath) {
			continue
		}

		localFV := e.state.Files[relPath]
		if localFV == nil {
			// We don't have this file -- request it unless it was deleted
			if !remoteFV.Deleted {
				go func(path string, pid peer.ID) {
					reqMsg, _ := EncodeMessage(MsgFileRequest, FileRequestPayload{Path: path})
					_ = e.node.SendToPeer(pid, reqMsg)
				}(relPath, from)
			}
			continue
		}

		order := CompareVersions(localFV.Versions, remoteFV.Versions)
		if order == RemoteNewer {
			go func(path string, pid peer.ID) {
				reqMsg, _ := EncodeMessage(MsgFileRequest, FileRequestPayload{Path: path})
				_ = e.node.SendToPeer(pid, reqMsg)
			}(relPath, from)
		}
	}
}

// reconcile scans all local files and updates the sync state for any that
// changed while Forge was not running. It is called once at engine start.
func (e *Engine) reconcile() {
	files, err := ScanSyncableFiles()
	if err != nil {
		log.Printf("sync: reconcile error: %v", err)
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	changed := false
	for relPath, info := range files {
		fv, ok := e.state.Files[relPath]
		if !ok || fv.Hash != info.Hash {
			e.state.RecordLocalChange(relPath, info.Hash, info.Modified, info.Size)
			changed = true
		}
	}

	if changed {
		_ = e.state.Save()
	}

	log.Printf("sync: reconciliation complete, %d files tracked", len(e.state.Files))
}

// GetSyncStatus returns a summary of the sync engine's current state,
// suitable for display in the UI.
func (e *Engine) GetSyncStatus() map[string]interface{} {
	e.mu.Lock()
	defer e.mu.Unlock()

	connectedCount := 0
	for _, p := range e.peers.Peers {
		pid, err := peer.Decode(p.ID)
		if err != nil {
			continue
		}
		if e.node.Connectedness(pid) == network.Connected {
			connectedCount++
		}
	}

	return map[string]interface{}{
		"peerId":         e.node.PeerID().String(),
		"humanId":        e.node.HumanID(),
		"totalPeers":     len(e.peers.Peers),
		"connectedPeers": connectedCount,
		"trackedFiles":   len(e.state.Files),
		"running":        true,
	}
}

// mergeCollections performs a union merge of two collections.json files,
// combining their collection arrays by ID so that no collection is
// duplicated. If either input cannot be parsed the other is returned
// as-is.
func mergeCollections(local, remote []byte) ([]byte, error) {
	var localStore, remoteStore struct {
		Collections []json.RawMessage `json:"collections"`
	}
	if err := json.Unmarshal(local, &localStore); err != nil {
		return remote, nil
	}
	if err := json.Unmarshal(remote, &remoteStore); err != nil {
		return local, nil
	}

	seen := map[string]bool{}
	var merged []json.RawMessage

	for _, c := range localStore.Collections {
		var obj map[string]interface{}
		_ = json.Unmarshal(c, &obj)
		if id, ok := obj["id"].(string); ok {
			seen[id] = true
		}
		merged = append(merged, c)
	}
	for _, c := range remoteStore.Collections {
		var obj map[string]interface{}
		_ = json.Unmarshal(c, &obj)
		if id, ok := obj["id"].(string); ok {
			if !seen[id] {
				merged = append(merged, c)
			}
		}
	}

	result := struct {
		Collections []json.RawMessage `json:"collections"`
	}{merged}
	return json.MarshalIndent(result, "", "  ")
}

// ---------------------------------------------------------------------------
// Manual sync methods
// ---------------------------------------------------------------------------

// SyncFile manually triggers sync for a specific file path. It reads the
// file, hashes it, updates the sync state, and broadcasts a FileAnnounce
// to all paired peers. The path should be a tilde-relative path (e.g.
// "~/.claude/agents/foo.md").
func (e *Engine) SyncFile(relPath string) error {
	absPath := resolveAbsPath(relPath)

	hash, err := HashFile(absPath)
	if err != nil {
		return fmt.Errorf("hash file %s: %w", relPath, err)
	}

	fi, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("stat file %s: %w", relPath, err)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.state.RecordLocalChange(relPath, hash, fi.ModTime().Unix(), fi.Size())
	_ = e.state.Save()

	fv := e.state.Files[relPath]
	msg, err := EncodeMessage(MsgFileAnnounce, FileAnnouncePayload{
		Path:     relPath,
		Hash:     hash,
		Versions: fv.Versions,
		Modified: fv.Modified,
		Size:     fv.Size,
	})
	if err != nil {
		return fmt.Errorf("encode announce for %s: %w", relPath, err)
	}
	e.node.BroadcastToPairedPeers(msg)
	return nil
}

// SyncFiles triggers sync for multiple paths. It processes each path
// independently, collecting errors. If any file fails to sync the
// combined error is returned.
func (e *Engine) SyncFiles(relPaths []string) error {
	var errs []string
	for _, rp := range relPaths {
		if err := e.SyncFile(rp); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("sync errors: %s", joinErrors(errs))
	}
	return nil
}

// GetManifest returns the current sync manifest.
func (e *Engine) GetManifest() *SyncManifest {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.manifest
}

// AddToManifest adds paths to the manifest and persists the change to disk.
func (e *Engine) AddToManifest(relPaths []string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.manifest.AddPaths(relPaths)
	return SaveManifest(e.manifest)
}

// RemoveFromManifest removes paths from the manifest and persists the
// change to disk.
func (e *Engine) RemoveFromManifest(relPaths []string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, rp := range relPaths {
		e.manifest.RemovePath(rp)
	}
	return SaveManifest(e.manifest)
}

// ---------------------------------------------------------------------------
// Conflict resolution
// ---------------------------------------------------------------------------

// GetConflicts returns a copy of all unresolved sync conflicts.
func (e *Engine) GetConflicts() []SyncConflict {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]SyncConflict, len(e.state.Conflicts))
	copy(out, e.state.Conflicts)
	return out
}

// ResolveConflict resolves the conflict for the given path using the
// specified resolution strategy:
//   - "keep-local": keep the local file unchanged, bump version vector
//   - "keep-remote": overwrite local with remote content, update version
//   - "keep-both": write remote content as a .conflict-<ts> file
//
// The conflict is removed from the state after resolution.
func (e *Engine) ResolveConflict(path string, resolution string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	idx := -1
	for i, c := range e.state.Conflicts {
		if c.Path == path {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("no conflict found for path: %s", path)
	}

	conflict := e.state.Conflicts[idx]
	absPath := resolveAbsPath(path)

	switch resolution {
	case "keep-local":
		// Keep the local file as-is; bump the local version vector so
		// the next sync broadcast propagates our decision.
		if fv, ok := e.state.Files[path]; ok {
			fv.Versions[e.state.LocalID]++
		}

	case "keep-remote":
		// Write the remote content to disk and update state.
		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			return fmt.Errorf("create dir for %s: %w", path, err)
		}
		if err := os.WriteFile(absPath, []byte(conflict.RemoteContent), 0644); err != nil {
			return fmt.Errorf("write remote content for %s: %w", path, err)
		}
		fi, err := os.Stat(absPath)
		if err != nil {
			return fmt.Errorf("stat %s after write: %w", path, err)
		}
		e.state.RecordLocalChange(path, conflict.RemoteHash, fi.ModTime().Unix(), fi.Size())

	case "keep-both":
		// Write the remote content as a conflict sidecar file.
		ext := filepath.Ext(absPath)
		base := absPath[:len(absPath)-len(ext)]
		ts := time.Now().Format("20060102-150405")
		conflictPath := fmt.Sprintf("%s.conflict-%s%s", base, ts, ext)
		if err := os.WriteFile(conflictPath, []byte(conflict.RemoteContent), 0644); err != nil {
			return fmt.Errorf("write conflict file for %s: %w", path, err)
		}
		// Bump local version so the original propagates
		if fv, ok := e.state.Files[path]; ok {
			fv.Versions[e.state.LocalID]++
		}

	default:
		return fmt.Errorf("unknown resolution: %s (expected keep-local, keep-remote, or keep-both)", resolution)
	}

	// Remove the resolved conflict from the list.
	e.state.Conflicts = append(e.state.Conflicts[:idx], e.state.Conflicts[idx+1:]...)
	_ = e.state.Save()

	return nil
}

// joinErrors concatenates a slice of error messages with "; " separators.
func joinErrors(errs []string) string {
	result := ""
	for i, e := range errs {
		if i > 0 {
			result += "; "
		}
		result += e
	}
	return result
}
