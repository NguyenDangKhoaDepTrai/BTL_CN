package server

import (
	"bufio"
	"crypto/sha1"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"tcp-app/torrent"
	"time"
)

// StartServer initializes the server to handle peer requests.
func StartServer(address string) error {
	listener, err := net.Listen("tcp", address) // Tạo socket server để lắng nghe trên cổng port
	if err != nil {
		return fmt.Errorf("error starting TCP server: %v", err)
	}
	defer listener.Close()

	fmt.Printf("Server listening on %s...\n", address)

	for {
		conn, err := listener.Accept() // Chấp nhận kết nối từ client
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}

		// Handle each connection in a new goroutine
		go handleConnection(conn)
	}
}

type TorrentFile struct {
	Announce    string
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
}

// FileWorker handles the file pieces for a specific torrent
type FileWorker struct {
	filePath    string
	pieces      [][]byte
	numPieces   int
	pieceHashes [][20]byte
}

// NewFileWorker creates and initializes a new FileWorker
func NewFileWorker(filePath string) (*FileWorker, error) {
	t := &TorrentFile{
		PieceLength: 256 * 1024, // 256KB pieces
	}
	// Get pieces using the StreamFilePieces function
	pieces, err := torrent.StreamFilePieces(filePath, t.PieceLength)
	if err != nil {
		return nil, fmt.Errorf("error streaming file pieces: %v", err)
	}

	// Calculate piece hashes
	pieceHashes := make([][20]byte, len(pieces))
	for i, piece := range pieces {
		pieceHashes[i] = sha1.Sum(piece)
	}

	return &FileWorker{
		filePath:    filePath,
		pieces:      pieces,
		numPieces:   len(pieces),
		pieceHashes: pieceHashes,
	}, nil
}

// Global map to store workers associated with info hashes
var connectionWorkers = make(map[string]*FileWorker)

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

func handleConnection(conn net.Conn) {
	// Add peer when connection starts
	peerManager.AddPeer(conn)

	// Remove peer when connection ends
	defer func() {
		peerManager.RemovePeer(conn)
		conn.Close()
	}()

	// Create a buffered reader to process incoming data
	reader := bufio.NewReader(conn)

	for {
		// Read client request
		message, err := reader.ReadString('\n')
		if err != nil {
			return // This will trigger the deferred RemovePeer
		}
		message = strings.TrimSpace(message)
		fmt.Printf("Received message: %s\n", message)

		// Process the message based on its type
		switch {
		case strings.HasPrefix(message, "test:"):
			fmt.Printf("Received test message: %s\n", message)
			conn.Write([]byte("OK\n"))
		case strings.HasPrefix(message, "HANDSHAKE:"):
			infoHash, worker := handleHandshake(conn, message)
			if worker == nil {
				return
			}
			// Store the worker in the global map using info hash
			connectionWorkers[infoHash] = worker

		case strings.HasPrefix(message, "Requesting"):
			parts := strings.Split(message, ":")
			infoHash := parts[1]
			worker, exists := connectionWorkers[infoHash]
			if !exists || worker == nil {
				conn.Write([]byte("ERROR: Handshake required\n"))
				continue
			}
			fmt.Printf("Received piece request: %s\n", message)
			handlePieceRequest(conn, message, worker)

		default:
			fmt.Printf("Unknown message: %s\n", message)
			conn.Write([]byte("ERROR: Unknown message\n"))
		}
	}
}

func handleHandshake(conn net.Conn, message string) (string, *FileWorker) {
	// Get the info hash from the message
	infoHashMessage := strings.TrimPrefix(message, "HANDSHAKE:")
	// Check if the info hash is in the torrent_info.json file
	torrentInfo, err := os.ReadFile("torrent_info.json")
	if err != nil {
		fmt.Printf("Error reading torrent_info.json: %v\n", err)
		return "", nil
	}
	var torrentInfoMap map[string]string
	err = json.Unmarshal(torrentInfo, &torrentInfoMap)
	if err != nil {
		fmt.Printf("Error unmarshalling torrent_info.json: %v\n", err)
		return "", nil
	}
	infoHash := torrentInfoMap["InfoHash"]
	if infoHash != infoHashMessage {
		fmt.Printf("Info hash mismatch: %s != %s\n", infoHash, infoHashMessage)
		return "", nil
	}
	filePath := torrentInfoMap["FilePath"]
	// Create worker for the file
	worker, err := NewFileWorker(filePath)
	if err != nil {
		fmt.Printf("Error creating file worker: %v\n", err)
		conn.Write([]byte("ERROR: Unable to process file\n"))
		return "", nil
	}

	conn.Write([]byte("OK\n"))
	return infoHash, worker
}

func handlePieceRequest(conn net.Conn, message string, worker *FileWorker) {
	parts := strings.Split(message, ":")
	if len(parts) != 3 {
		conn.Write([]byte("ERROR: Invalid request format\n"))
		return
	}

	index := strings.TrimSpace(parts[2])
	pieceIndex, err := strconv.Atoi(index)
	if err != nil || pieceIndex < 0 || pieceIndex >= worker.numPieces {
		conn.Write([]byte("ERROR: Invalid piece index\n"))
		return
	}

	// First send the piece size as a fixed-length header (8 bytes)
	pieceSize := len(worker.pieces[pieceIndex])
	sizeHeader := make([]byte, 8)
	binary.BigEndian.PutUint64(sizeHeader, uint64(pieceSize))

	// Send size header followed by piece data
	conn.Write(sizeHeader)
	conn.Write(worker.pieces[pieceIndex])
}
