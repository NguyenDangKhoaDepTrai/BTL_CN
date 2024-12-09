package server

import (
	"bufio"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"tcp-app/torrent"
)

// StartServer initializes the server to handle peer requests.
func StartServer(serverAddress string) error {
	listener, err := net.Listen("tcp", serverAddress)
	if err != nil {
		return fmt.Errorf("error starting TCP server: %v", err)
	}
	defer listener.Close()

	fmt.Printf("Server listening on %s...\n", serverAddress)
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

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Create a buffered reader to process incoming data
	reader := bufio.NewReader(conn)

	for {
		// Read client request
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading from connection: %v\n", err)
			return
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

// ListTorrentFiles returns a list of all .torrent files in the torrent_files directory
func ListTorrentFiles() ([]string, error) {
	files, err := os.ReadDir("torrent_files")
	if err != nil {
		return nil, fmt.Errorf("failed to read torrent_files directory: %v", err)
	}

	var torrentFiles []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".torrent") {
			torrentFiles = append(torrentFiles, file.Name())
		}
	}
	return torrentFiles, nil
}

func ParseTorrentFile(torrent_file_name string) ([]torrent.TorrentFile, error) {
	torrentPath := "torrent_files/" + torrent_file_name
	tfs, err := torrent.Open(torrentPath)
	if err != nil {
		return nil, fmt.Errorf("error opening torrent file: %v", err)
	}
	return tfs, nil
}

func handleHandshake(conn net.Conn, message string) (string, *FileWorker) {
	// Get the info hash from the message
	torrent_file_name := strings.TrimPrefix(message, "HANDSHAKE:")
	//Check if the infohash is in the list of torrent files
	torrentFiles, err := ListTorrentFiles()
	if err != nil {
		fmt.Printf("Error listing torrent files: %v\n", err)
		return "", nil
	}
	found := false
	for _, file := range torrentFiles {
		if strings.TrimSuffix(file, ".torrent") == torrent_file_name {
			found = true
			break
		}
	}
	if !found {
		fmt.Printf("This torrent file is not in the list of torrent files currently available\n")
		return "", nil
	}

	// Create worker for the file
	tfs, err := ParseTorrentFile(torrent_file_name + ".torrent")
	if err != nil {
		fmt.Printf("Error parsing torrent file: %v\n", err)
	}
	for _, tf := range tfs {
		fmt.Printf("Downloading file: %v\n", tf.Name)
	}
	fileName := "files/" + tfs[0].Name
	worker, err := NewFileWorker(fileName)
	if err != nil {
		fmt.Printf("Error creating file worker: %v\n", err)
		conn.Write([]byte("ERROR: Unable to process file\n"))
		return "", nil
	}

	conn.Write([]byte("OK\n"))
	return torrent_file_name, worker
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
