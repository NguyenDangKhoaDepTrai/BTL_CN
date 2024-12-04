package main

// Add to imports if not present
import (
	"fmt"
	"net"
	"strconv"
	"strings"
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
func (pm *PeerManager) AddPeer(conn net.Conn) error {
	addr := conn.RemoteAddr().(*net.TCPAddr)
	peerAddr := fmt.Sprintf("%s", addr.IP.String())

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if peer with same IP already exists
	for existingAddr, existingPeer := range pm.peers {
		if existingPeer.IP == addr.IP.String() {
			fmt.Printf("\n[%s] Peer %s is already connected (as %s)\n",
				time.Now().Format("2006-01-02 15:04:05"),
				addr.IP.String(),
				existingAddr)
			return nil
		}
	}

	// Add peer to the manager
	pm.peers[peerAddr] = &PeerConnection{
		IP:          addr.IP.String(),
		Port:        strconv.Itoa(addr.Port),
		ConnectedAt: time.Now(),
	}

	// Notify about new connection
	fmt.Printf("\n[%s] New peer connected from %s\n", time.Now().Format("2006-01-02 15:04:05"), addr.IP.String())
	fmt.Printf("Total connected peers: %d\n", len(pm.peers))
	return nil
}

// RemovePeer removes a peer and notifies about the disconnection
func (pm *PeerManager) RemovePeer(conn net.Conn) error {
	addr := conn.RemoteAddr().(*net.TCPAddr)
	peerAddr := fmt.Sprintf("%s", addr.IP.String())

	pm.mu.Lock()
	defer pm.mu.Unlock()

	if peer, exists := pm.peers[peerAddr]; exists {
		duration := time.Since(peer.ConnectedAt).Round(time.Second)
		delete(pm.peers, peerAddr)

		fmt.Printf("\n[%s] Peer disconnected from %s (connected for %s)\n", time.Now().Format("2006-01-02 15:04:05"), peerAddr, duration)
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
		duration := time.Since(peer.ConnectedAt).Round(time.Second)
		peerInfo := fmt.Sprintf("%s:%s (connected for %s)",
			peer.IP,
			peer.Port,
			duration)
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

	// Handle different commands
	switch {
	case strings.HasPrefix(data, "START:"):
		err = peerManager.AddPeer(conn)
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

func getLocalIP() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "unknown"
	}

	for _, iface := range interfaces {
		// Check if it's a wireless interface (common naming patterns)
		if strings.Contains(strings.ToLower(iface.Name), "wi-fi") ||
			strings.Contains(strings.ToLower(iface.Name), "wlan") {

			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}

			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok {
					if ip4 := ipnet.IP.To4(); ip4 != nil {
						return ip4.String()
					}
				}
			}
		}
	}
	return "unknown"
}

func main() {
	// Khởi tạo server
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Printf("Failed to initialize tracker: %v\n", err)
		return
	}
	defer listener.Close()

	Address := getLocalIP() + ":8080"
	fmt.Printf("[%s] Tracker is running at address: %s\n", time.Now().Format("2006-01-02 15:04:05"), Address)

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
