package main

// Add to imports if not present
import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// PeerConnection represents a connected peer
type PeerConnection struct {
	IP       string
	Port     string
	Filename string
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
func (pm *PeerManager) AddPeer(conn net.Conn, peerAddr string, fileName string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if peer with same IP already exists
	for existingAddr, existingPeer := range pm.peers {
		if existingPeer.IP == peerAddr {
			fmt.Printf("\n[%s] Peer %s is already connected (as %s)\n",
				time.Now().Format("2006-01-02 15:04:05"),
				peerAddr,
				existingAddr)
			return nil
		}
	}

	// Add peer to the manager
	pm.peers[peerAddr] = &PeerConnection{
		Filename: fileName,
		IP:       peerAddr,
		Port:     ":8080",
	}

	// Notify about new connection
	fmt.Printf("\n[%s] New peer connected from %s\n", time.Now().Format("2006-01-02 15:04:05"), peerAddr)
	fmt.Printf("Total connected peers: %d\n", len(pm.peers))
	return nil
}

// RemovePeer removes a peer and notifies about the disconnection
func (pm *PeerManager) RemovePeer(conn net.Conn) error {
	addr := conn.RemoteAddr().(*net.TCPAddr)
	peerAddr := fmt.Sprintf("%s", addr.IP.String())

	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.peers[peerAddr]; exists {
		delete(pm.peers, peerAddr)

		fmt.Printf("\n[%s] Peer disconnected from %s\n", time.Now().Format("2006-01-02 15:04:05"), peerAddr)
		fmt.Printf("Total connected peers: %d\n", len(pm.peers))
	}
	return nil
}

// GetConnectedPeers returns a list of peer details including IP, Port, and connection duration
func (pm *PeerManager) GetConnectedPeers() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	peers := make([]string, 0, len(pm.peers))
	for _, peer := range pm.peers {
		peerInfo := fmt.Sprintf("%s:%s",
			peer.IP,
			peer.Port,
		)
		peers = append(peers, peerInfo)
	}
	return peers
}

// HandleConnection xử lý kết nối từ peer
func handleConnection(conn net.Conn) {
	// Create a buffer to store incoming data
	buffer := make([]byte, 1024)
	// Read data from the connection
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Printf("Error reading data from peer: %v\n", err)
		return
	}

	// Convert the data to a string and print it
	data := string(buffer[:n])
	fmt.Printf("Received data from peer: %s\n", data)

	args := strings.Split(data, ":")
	peerAddr := args[1]
	fileName := args[2]

	// Handle different commands
	switch {
	case strings.HasPrefix(data, "START:"):
		err = peerManager.AddPeer(conn, peerAddr, fileName)
		if err != nil {
			fmt.Printf("Error adding peer: %v\n", err)
			return
		}
	case strings.HasPrefix(data, "STOP:"):
		err = peerManager.RemovePeer(conn)
		if err != nil {
			fmt.Printf("Error removing peer: %v\n", err)
			return
		}
	}

	// Prepare response with connected peers information
	peers := peerManager.GetConnectedPeers()
	response := fmt.Sprintf("Connected peers (%d):\n%s",
		len(peers),
		strings.Join(peers, "\n")+"!")

	// Send response to peer
	_, err = conn.Write([]byte(response))
	fmt.Printf("Sent response to peer: %s\n----------------------------------------------- \n", response)
	if err != nil {
		fmt.Printf("Error sending response to peer: %v\n", err)
		return
	}
}

func main() {
	trackerAddress := "192.168.101.99:8080"
	// Khởi tạo server
	listener, err := net.Listen("tcp", trackerAddress)
	if err != nil {
		fmt.Printf("Failed to initialize tracker: %v\n", err)
		return
	}
	defer listener.Close()

	fmt.Printf("[%s] Tracker is running at address: %s\n", time.Now().Format("2006-01-02 15:04:05"), trackerAddress)

	// Chấp nhận kết nối
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Failed to accept connection: %v\n", err)
			continue
		}
		go handleConnection(conn)
	}
}
