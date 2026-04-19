package sync

import (
	"context"
	"fmt"
	"io"
	"log"
	gosync "sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
)

// SyncEngine is the interface that the Engine must implement so that the Node
// can delegate peer-management and message-handling decisions. This interface
// is defined here to break the compile-time dependency on Engine, which will be
// created in engine.go.
type SyncEngine interface {
	// IsPairedPeer reports whether the given peer ID belongs to a device the
	// user has explicitly paired with.
	IsPairedPeer(pid peer.ID) bool
	// GetPairedPeers returns information about every paired peer.
	GetPairedPeers() []PeerInfo
	// HandleIncoming processes raw inbound bytes received from the identified
	// peer over the given stream.
	HandleIncoming(from peer.ID, data []byte, s network.Stream)
}

// DiscoveryEvent describes a peer discovery or connection change that the UI
// or other subsystems may want to react to.
type DiscoveryEvent struct {
	PeerID  string `json:"peerId"`
	HumanID string `json:"humanId"`
	Event   string `json:"event"` // "discovered", "connected", "disconnected"
}

// OnDiscoveryFunc is a callback signature for peer discovery and connection
// events. Implementations must be safe for concurrent use.
type OnDiscoveryFunc func(event DiscoveryEvent)

// Node wraps a libp2p host together with mDNS and DHT discovery services.
// It implements the NodeHost interface defined in sync.go so it can be used
// by the Engine for network operations. It also provides protocol registration
// and peer discovery.
type Node struct {
	Host   host.Host
	engine SyncEngine

	dht       *dht.IpfsDHT
	mdnsSvc   mdns.Service
	routingD  *drouting.RoutingDiscovery
	onDisc    OnDiscoveryFunc
	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce gosync.Once
}

// Compile-time check: Node must satisfy the NodeHost interface.
var _ NodeHost = (*Node)(nil)

// NewNode creates a libp2p host with TCP and QUIC transports, NAT port
// mapping, and hole punching enabled. The host identity is loaded from
// ~/.agentdesk/identity.key (or created on first run). The returned Node is ready
// for discovery but the SyncEngine must be attached separately via SetEngine
// before starting discovery.
func NewNode() (*Node, error) {
	priv, err := LoadOrCreateIdentity()
	if err != nil {
		return nil, fmt.Errorf("load identity: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	h, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/0",
			"/ip6/::/tcp/0",
			"/ip4/0.0.0.0/udp/0/quic-v1",
			"/ip6/::/udp/0/quic-v1",
		),
		libp2p.NATPortMap(),
		libp2p.EnableHolePunching(),
		libp2p.EnableRelay(),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create libp2p host: %w", err)
	}

	log.Printf("[node] libp2p host started: %s", h.ID())
	for _, addr := range h.Addrs() {
		log.Printf("[node]   listening on %s/p2p/%s", addr, h.ID())
	}

	return &Node{
		Host:   h,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// SetEngine attaches the SyncEngine that handles message routing and peer
// management decisions. This must be called before StartDiscovery.
func (n *Node) SetEngine(e SyncEngine) {
	n.engine = e
}

// SetOnDiscovery registers a callback that will be invoked for peer discovery
// and connection events. Pass nil to clear the callback.
func (n *Node) SetOnDiscovery(fn OnDiscoveryFunc) {
	n.onDisc = fn
}

// ---------------------------------------------------------------------------
// NodeHost interface implementation
// ---------------------------------------------------------------------------

// PeerID returns the local peer ID of the underlying libp2p host.
func (n *Node) PeerID() peer.ID {
	return n.Host.ID()
}

// HumanID returns the human-readable identifier for the local peer.
func (n *Node) HumanID() string {
	return HumanReadableID(n.Host.ID())
}

// Connectedness reports the connection state for a given peer.
func (n *Node) Connectedness(pid peer.ID) network.Connectedness {
	return n.Host.Network().Connectedness(pid)
}

// ConnectToPeer actively tries to connect to a peer by ID. It uses the
// DHT to find the peer's addresses if they're not already known. This is
// called after pairing to establish the initial connection.
func (n *Node) ConnectToPeer(pid peer.ID) {
	// Check if already connected
	if n.Host.Network().Connectedness(pid) == network.Connected {
		log.Printf("[node] already connected to %s", pid)
		n.emitDiscovery(pid, "connected")
		n.requestFullSync(pid)
		return
	}

	log.Printf("[node] attempting to connect to peer %s...", pid)

	// First try direct connection if we have addresses in the peerstore
	ctx, cancel := context.WithTimeout(n.ctx, 15*time.Second)
	err := n.Host.Connect(ctx, peer.AddrInfo{ID: pid})
	cancel()
	if err == nil {
		log.Printf("[node] connected to peer %s (direct)", pid)
		n.emitDiscovery(pid, "connected")
		n.requestFullSync(pid)
		return
	}

	// Try DHT lookup to find the peer's addresses
	if n.dht != nil {
		log.Printf("[node] looking up peer %s via DHT...", pid)
		ctx2, cancel2 := context.WithTimeout(n.ctx, 30*time.Second)
		pi, err := n.dht.FindPeer(ctx2, pid)
		cancel2()
		if err == nil && len(pi.Addrs) > 0 {
			ctx3, cancel3 := context.WithTimeout(n.ctx, 15*time.Second)
			err = n.Host.Connect(ctx3, pi)
			cancel3()
			if err == nil {
				log.Printf("[node] connected to peer %s (via DHT)", pid)
				n.emitDiscovery(pid, "connected")
				n.requestFullSync(pid)
				return
			}
		}
	}

	log.Printf("[node] could not connect to peer %s — will retry on discovery", pid)
}

// requestFullSync sends a SyncRequest to a connected peer to exchange
// full file state. This triggers the remote peer to send back its complete
// file list, allowing both sides to identify and transfer missing files.
func (n *Node) requestFullSync(pid peer.ID) {
	msg, err := EncodeMessage(MsgSyncRequest, struct{}{})
	if err != nil {
		return
	}
	if err := n.SendToPeer(pid, msg); err != nil {
		log.Printf("[node] failed to request sync from %s: %v", pid, err)
	} else {
		log.Printf("[node] requested full sync from %s", pid)
	}
}

// SendToPeer sends raw message bytes to a specific peer by opening a new
// stream on the Forge sync protocol. The stream is closed after the bytes
// are written.
func (n *Node) SendToPeer(pid peer.ID, data []byte) error {
	ctx, cancel := context.WithTimeout(n.ctx, 30*time.Second)
	defer cancel()

	s, err := n.Host.NewStream(ctx, pid, protocol.ID(ProtocolID))
	if err != nil {
		return fmt.Errorf("open stream to %s: %w", pid, err)
	}
	defer s.Close()

	if _, err := s.Write(data); err != nil {
		return fmt.Errorf("write to %s: %w", pid, err)
	}
	return nil
}

// BroadcastToPairedPeers sends raw message bytes to every paired peer that is
// currently connected. Errors for individual peers are logged but do not
// prevent delivery to other peers.
func (n *Node) BroadcastToPairedPeers(data []byte) {
	if n.engine == nil {
		return
	}

	for _, pi := range n.engine.GetPairedPeers() {
		pid, err := peer.Decode(pi.ID)
		if err != nil {
			log.Printf("[node] invalid peer ID %q in paired list: %v", pi.ID, err)
			continue
		}

		if n.Host.Network().Connectedness(pid) != network.Connected {
			continue
		}

		if err := n.SendToPeer(pid, data); err != nil {
			log.Printf("[node] broadcast to %s failed: %v", pid, err)
		}
	}
}

// EmitEvent notifies the frontend of a sync-related event. This is a
// placeholder implementation; the actual Wails event emission will be wired
// in the Engine or app layer.
func (n *Node) EmitEvent(event string, data map[string]string) {
	log.Printf("[node] event %s: %v", event, data)
}

// ---------------------------------------------------------------------------
// Protocol handling
// ---------------------------------------------------------------------------

// RegisterProtocol sets up the stream handler for the Forge sync protocol so
// the node can receive inbound sync messages from paired peers.
func (n *Node) RegisterProtocol() {
	n.Host.SetStreamHandler(protocol.ID(ProtocolID), func(s network.Stream) {
		defer s.Close()

		remotePeer := s.Conn().RemotePeer()

		data, err := io.ReadAll(io.LimitReader(s, 10<<20)) // 10 MiB max
		if err != nil {
			log.Printf("[node] error reading stream from %s: %v", remotePeer, err)
			return
		}

		if n.engine != nil {
			n.engine.HandleIncoming(remotePeer, data, s)
		}
	})
}

// ---------------------------------------------------------------------------
// Discovery
// ---------------------------------------------------------------------------

// StartDiscovery starts both mDNS (LAN) and DHT (WAN) peer discovery. It
// connects to IPFS bootstrap nodes for DHT operation and begins advertising
// the Forge rendezvous namespace. Discovered peers that are in the paired-peer
// list are connected automatically.
func (n *Node) StartDiscovery() error {
	if err := n.startDHT(); err != nil {
		return fmt.Errorf("start DHT: %w", err)
	}

	if err := n.startMDNS(); err != nil {
		log.Printf("[node] mDNS start failed (LAN discovery disabled): %v", err)
		// mDNS failure is non-fatal; WAN discovery via DHT still works.
	}

	go n.dhtDiscoveryLoop()

	return nil
}

// startDHT initializes the Kademlia DHT in client mode using the IPFS
// bootstrap peers and performs an initial bootstrap.
func (n *Node) startDHT() error {
	bootstrapPeers := dht.GetDefaultBootstrapPeerAddrInfos()

	kademliaDHT, err := dht.New(n.ctx, n.Host,
		dht.Mode(dht.ModeClient),
		dht.BootstrapPeers(bootstrapPeers...),
	)
	if err != nil {
		return fmt.Errorf("create DHT: %w", err)
	}

	if err := kademliaDHT.Bootstrap(n.ctx); err != nil {
		return fmt.Errorf("bootstrap DHT: %w", err)
	}

	// Connect to bootstrap peers concurrently.
	var wg gosync.WaitGroup
	for _, pi := range bootstrapPeers {
		wg.Add(1)
		go func(pi peer.AddrInfo) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(n.ctx, 15*time.Second)
			defer cancel()
			if err := n.Host.Connect(ctx, pi); err != nil {
				log.Printf("[node] bootstrap peer %s unreachable: %v", pi.ID, err)
			}
		}(pi)
	}
	wg.Wait()

	n.dht = kademliaDHT
	n.routingD = drouting.NewRoutingDiscovery(kademliaDHT)
	return nil
}

// startMDNS initializes mDNS-based LAN discovery using the Forge service tag.
func (n *Node) startMDNS() error {
	svc := mdns.NewMdnsService(n.Host, "agentdesk-sync", &mdnsNotifee{node: n})
	if err := svc.Start(); err != nil {
		return fmt.Errorf("mdns start: %w", err)
	}
	n.mdnsSvc = svc
	return nil
}

// dhtDiscoveryLoop periodically advertises this node under the Forge
// rendezvous namespace and searches for other providers. Discovered peers
// that are paired are connected automatically.
func (n *Node) dhtDiscoveryLoop() {
	const rendezvous = "agentdesk/sync/v1"

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Run immediately on first call, then on every tick.
	n.dhtDiscoverOnce(rendezvous)
	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.dhtDiscoverOnce(rendezvous)
		}
	}
}

// dhtDiscoverOnce performs a single advertise + find-peers round via the DHT
// routing discovery layer.
func (n *Node) dhtDiscoverOnce(rendezvous string) {
	if n.routingD == nil {
		return
	}

	advCtx, advCancel := context.WithTimeout(n.ctx, 30*time.Second)
	defer advCancel()

	if _, err := n.routingD.Advertise(advCtx, rendezvous); err != nil {
		log.Printf("[node] DHT advertise error: %v", err)
	}

	findCtx, findCancel := context.WithTimeout(n.ctx, 30*time.Second)
	defer findCancel()

	peerChan, err := n.routingD.FindPeers(findCtx, rendezvous)
	if err != nil {
		log.Printf("[node] DHT find peers error: %v", err)
		return
	}

	for pi := range peerChan {
		if pi.ID == n.Host.ID() || len(pi.Addrs) == 0 {
			continue
		}
		n.handleDiscoveredPeer(pi)
	}
}

// handleDiscoveredPeer is called for every peer found through either mDNS or
// DHT discovery. It emits a discovery event and, if the peer is paired,
// attempts a connection.
func (n *Node) handleDiscoveredPeer(pi peer.AddrInfo) {
	n.emitDiscovery(pi.ID, "discovered")

	if n.engine == nil {
		return
	}

	if n.Host.Network().Connectedness(pi.ID) == network.Connected {
		return // already connected
	}

	// Connect to paired peers automatically.
	// Also connect to unpaired peers — they might be trying to pair with us.
	// The protocol handler will auto-pair them if they send a valid message.
	if !n.engine.IsPairedPeer(pi.ID) {
		// Store their addresses so they can connect to us
		n.Host.Peerstore().AddAddrs(pi.ID, pi.Addrs, time.Hour)
		return
	}

	ctx, cancel := context.WithTimeout(n.ctx, 15*time.Second)
	defer cancel()

	if err := n.Host.Connect(ctx, pi); err != nil {
		log.Printf("[node] failed to connect to paired peer %s: %v", pi.ID, err)
		return
	}

	log.Printf("[node] connected to paired peer %s", pi.ID)
	n.emitDiscovery(pi.ID, "connected")

	// Request full sync state from the newly connected peer
	// so we can exchange any files that differ.
	go func() {
		msg, err := EncodeMessage(MsgSyncRequest, struct{}{})
		if err != nil {
			return
		}
		if err := n.SendToPeer(pi.ID, msg); err != nil {
			log.Printf("[node] failed to request sync state from %s: %v", pi.ID, err)
		} else {
			log.Printf("[node] requested full sync state from %s", pi.ID)
		}
	}()
}

// emitDiscovery safely invokes the discovery callback if one is registered.
func (n *Node) emitDiscovery(pid peer.ID, event string) {
	if n.onDisc != nil {
		n.onDisc(DiscoveryEvent{
			PeerID:  pid.String(),
			HumanID: HumanReadableID(pid),
			Event:   event,
		})
	}
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

// Stop gracefully shuts down the node, closing discovery services, the DHT,
// and the libp2p host. It is safe to call Stop multiple times.
func (n *Node) Stop() error {
	var stopErr error
	n.closeOnce.Do(func() {
		n.cancel()

		if n.mdnsSvc != nil {
			if err := n.mdnsSvc.Close(); err != nil {
				log.Printf("[node] mDNS close error: %v", err)
				stopErr = err
			}
		}

		if n.dht != nil {
			if err := n.dht.Close(); err != nil {
				log.Printf("[node] DHT close error: %v", err)
				stopErr = err
			}
		}

		if err := n.Host.Close(); err != nil {
			log.Printf("[node] host close error: %v", err)
			stopErr = err
		}

		log.Println("[node] stopped")
	})
	return stopErr
}

// ---------------------------------------------------------------------------
// mDNS notifee
// ---------------------------------------------------------------------------

// mdnsNotifee implements the mdns.Notifee interface so the Node can react to
// peers discovered on the local network.
type mdnsNotifee struct {
	node *Node
}

// HandlePeerFound is called by the mDNS service when a new peer is discovered
// on the LAN. It delegates to the shared discovery handler.
func (m *mdnsNotifee) HandlePeerFound(pi peer.AddrInfo) {
	m.node.handleDiscoveredPeer(pi)
}
