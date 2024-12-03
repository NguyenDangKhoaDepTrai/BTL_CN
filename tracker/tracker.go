package main

// Add to imports if not present
import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)

// PeerConnection represents a connected peer
type PeerConnection struct {
	IP          string
	Port        string
	ConnectedAt time.Time
}

// PeerManager manages connected peers
type PeerManager struct {
	peers map[string]*PeerConnection
	mu    sync.RWMutex
}

// Create a global peer manager
var peerManager = &PeerManager{
	peers: make(map[string]*PeerConnection),
}

// AddPeer adds a new peer to the manager and notifies about the connection
func (pm *PeerManager) AddPeer(conn net.Conn) {
	addr := conn.RemoteAddr().(*net.TCPAddr)
	peerAddr := fmt.Sprintf("%s:%d", addr.IP.String(), addr.Port)

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Add peer to the manager
	pm.peers[peerAddr] = &PeerConnection{
		IP:          addr.IP.String(),
		Port:        strconv.Itoa(addr.Port),
		ConnectedAt: time.Now(),
	}

	// Notify about new connection
	fmt.Printf("\n[%s] New peer connected from %s\n",
		time.Now().Format("2006-01-02 15:04:05"),
		peerAddr)

	// Print current number of connected peers
	fmt.Printf("Total connected peers: %d\n", len(pm.peers))
}

// RemovePeer removes a peer and notifies about the disconnection
func (pm *PeerManager) RemovePeer(conn net.Conn) {
	addr := conn.RemoteAddr().(*net.TCPAddr)
	peerAddr := fmt.Sprintf("%s:%d", addr.IP.String(), addr.Port)

	pm.mu.Lock()
	defer pm.mu.Unlock()

	if peer, exists := pm.peers[peerAddr]; exists {
		duration := time.Since(peer.ConnectedAt).Round(time.Second)
		delete(pm.peers, peerAddr)

		fmt.Printf("\n[%s] Peer disconnected from %s (connected for %s)\n",
			time.Now().Format("2006-01-02 15:04:05"),
			peerAddr,
			duration)
		fmt.Printf("Total connected peers: %d\n", len(pm.peers))
	}
}

// GetConnectedPeers returns a list of currently connected peers
func (pm *PeerManager) GetConnectedPeers() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	peers := make([]string, 0, len(pm.peers))
	for addr := range pm.peers {
		peers = append(peers, addr)
	}
	return peers
}

// HandleConnection xử lý kết nối từ peer
func handleConnection(conn net.Conn) {
	defer conn.Close()
	peerManager.AddPeer(conn)
	defer peerManager.RemovePeer(conn)

	// Giữ kết nối mở
	buffer := make([]byte, 1024)
	for {
		_, err := conn.Read(buffer)
		if err != nil {
			return
		}
	}
}

func main() {
	// Khởi tạo server
	listener, err := net.Listen("tcp", ":6881")
	if err != nil {
		fmt.Printf("Không thể khởi tạo tracker: %v\n", err)
		return
	}
	defer listener.Close()

	fmt.Printf("[%s] Tracker đang chạy tại địa chỉ: %s\n",
		time.Now().Format("2006-01-02 15:04:05"),
		listener.Addr().String())

	// Chấp nhận kết nối
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Lỗi khi chấp nhận kết nối: %v\n", err)
			continue
		}
		go handleConnection(conn)
	}
}
