package client

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"tcp-app/torrent"
)

type AddrAndFilename struct {
	Addr     string
	Filename string
}

var connectedTrackerAddresses []AddrAndFilename

type PieceWork struct {
	Index int
	Hash  []byte
	Size  int64
}

type PieceResult struct {
	Index int
	Data  []byte
	Error error
}

// TorrentInfo struct để parse JSON
type TorrentInfo struct {
	FileName    string `json:"FileName"`
	FilePath    string `json:"FilePath"`
	InfoHash    string `json:"InfoHash"`
	PieceHashes string `json:"PieceHashes"`
}

func StartDownload(torrentFile string, anotherPeerAddress []string, peerAddress string) {
	fmt.Println("Starting download for:", torrentFile)

	// Parse torrent file using the torrent package
	tfs, err := torrent.Open("torrent_files/" + torrentFile)
	if err != nil {
		fmt.Printf("Error opening torrent file: %v\n", err)
		return
	}

	// Mock the list of peers
	peers := anotherPeerAddress

	// First, test connection and handshake with peers
	var activePeers []string
	for _, peer := range peers {
		fmt.Printf("Testing connection to peer: %s\n", peer)
		err := TestConnection(peer)
		if err != nil {
			fmt.Printf("Peer %s is not available: %v\n", peer, err)
			continue
		}
		fmt.Printf("Peer %s is available\n", peer)
		// Use the first file's InfoHash for handshake (assuming all files share the same InfoHash)
		if err := performHandshake(peer, tfs[0].InfoHash[:]); err != nil {
			fmt.Printf("Handshake failed with peer %s: %v\n", peer, err)
			continue
		}
		activePeers = append(activePeers, peer)
	}

	if len(activePeers) == 0 {
		fmt.Println("No available active peers found!")
		return
	}

	// Process each file in the torrent
	for _, tf := range tfs {
		fmt.Printf("Downloading file: %s\n", tf.Name)

		// Create channels for the worker pool
		const numWorkers = 3
		workQueue := make(chan PieceWork, len(tf.PieceHashes))
		results := make(chan PieceResult, len(tf.PieceHashes))

		// Enqueue work for current file
		for i, hash := range tf.PieceHashes {
			workQueue <- PieceWork{
				Index: i,
				Hash:  hash[:],
				Size:  int64(tf.PieceLength),
			}
		}
		close(workQueue)

		// Start workers for current file
		var wg sync.WaitGroup
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			peerIndex := i % len(activePeers)
			go func(peer string) {
				defer wg.Done()
				downloadWorker(peer, workQueue, results, tf.InfoHash[:])
			}(activePeers[peerIndex])
		}

		// Wait for workers to complete
		go func() {
			wg.Wait()
			close(results)
		}()

		// Collect results and merge file
		piecesByIndex := make(map[int][]byte)
		for result := range results {
			if result.Error != nil {
				fmt.Printf("Error downloading piece %d: %v\n", result.Index, result.Error)
				continue
			}
			piecesByIndex[result.Index] = result.Data

			calculatedHash := sha1.Sum(result.Data)
			if !bytes.Equal(calculatedHash[:], tf.PieceHashes[result.Index][:]) {
				fmt.Printf("Piece %d hash mismatch!\n", result.Index)
			}
			fmt.Printf("Successfully downloaded piece %d of %s\n", result.Index, tf.Name)
		}

		// Merge pieces into current file
		if err := tf.MergePieces(tf.Name, piecesByIndex); err != nil {
			fmt.Printf("Error merging pieces for %s: %v\n", tf.Name, err)
			continue // Continue with next file even if current fails
		}

		fmt.Printf("Download complete for file: %s\n", tf.Name)
		trackerAddress := tf.Announce
		torrent.Create([]string{tf.Name}, trackerAddress)

		err = ConnectToTracker(trackerAddress, peerAddress, tf.Name)
		if err != nil {
			fmt.Printf("Failed to connect to tracker: %v\n", err)
		}
		exist := false
		for _, tracker := range connectedTrackerAddresses {
			if tracker.Addr == trackerAddress && tracker.Filename == tf.Name {
				exist = true
				break
			}
		}
		if !exist {
			connectedTrackerAddresses = append(connectedTrackerAddresses, AddrAndFilename{Addr: trackerAddress, Filename: tf.Name})
		}
	}

	fmt.Println("All downloads complete!")
}

func downloadWorker(peer string, work <-chan PieceWork, results chan<- PieceResult, infoHash []byte) {
	for piece := range work {
		fmt.Printf("Downloading piece %d from peer %s\n", piece.Index, peer)
		data, err := requestPieceFromPeer(peer, piece.Index, infoHash)

		results <- PieceResult{
			Index: piece.Index,
			Data:  data,
			Error: err,
		}
	}
}

func requestPieceFromPeer(address string, pieceIndex int, infoHash []byte) ([]byte, error) {
	conn, err := net.DialTimeout("tcp", address, 60*time.Second)
	if err != nil {
		return nil, fmt.Errorf("error connecting to peer: %v", err)
	}
	defer conn.Close()

	// First perform handshake if not already done
	if err := performHandshake(address, infoHash); err != nil {
		return nil, fmt.Errorf("handshake failed: %v", err)
	}

	// Request the piece
	message := fmt.Sprintf("Requesting:%x:%d\n", infoHash, pieceIndex)
	if _, err := conn.Write([]byte(message)); err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}

	// Read the piece size first (8 bytes)
	sizeHeader := make([]byte, 8)
	if _, err := io.ReadFull(conn, sizeHeader); err != nil {
		return nil, fmt.Errorf("error reading piece size: %v", err)
	}
	pieceSize := binary.BigEndian.Uint64(sizeHeader)

	// Read the exact number of bytes for the piece
	data := make([]byte, pieceSize)
	if _, err := io.ReadFull(conn, data); err != nil {
		return nil, fmt.Errorf("error reading piece data: %v", err)
	}

	return data, nil
}

func TestConnection(address string) error {
	// Set timeout for the entire operation
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return fmt.Errorf("connection failed: %v", err)
	}
	defer conn.Close()

	// Set read/write deadlines
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Send a test message
	_, err = conn.Write([]byte("test:\n")) // Add newline as message delimiter
	if err != nil {
		return fmt.Errorf("failed to send test message: %v", err)
	}

	// Read response
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	fmt.Printf("Received response: %s", response)
	return nil
}

func performHandshake(address string, infoHash []byte) error {
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return fmt.Errorf("handshake connection failed: %v", err)
	}
	defer conn.Close()

	// Send handshake message
	handshakeMsg := fmt.Sprintf("HANDSHAKE:%x\n", infoHash)
	if _, err := conn.Write([]byte(handshakeMsg)); err != nil {
		return fmt.Errorf("failed to send handshake: %v", err)
	}

	// Read handshake response
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read handshake response: %v", err)
	}

	if response != "OK\n" {
		return fmt.Errorf("invalid handshake response: %s", response)
	}

	return nil
}
func AnnounceToTracker(peerAddress string, torrentFilename string) error {
	tfs, err := torrent.Open("torrent_files/" + torrentFilename)
	if err != nil {
		fmt.Printf("Error opening torrent file: %v\n", err)
		return err
	}
	trackerAddress := ""
	filename := ""
	for _, tf := range tfs {
		trackerAddress = tf.Announce
		filename = tf.Name
		err := ConnectToTracker(trackerAddress, peerAddress, filename)
		if err != nil {
			fmt.Printf("Failed to connect to tracker %s for file %s: (error %v)\n", trackerAddress, filename, err)
			return err
		}
		exist := false
		for _, tracker := range connectedTrackerAddresses {
			if tracker.Addr == trackerAddress && tracker.Filename == filename {
				fmt.Println("You are already connected to this tracker for this file")
				exist = true
				break
			}
		}
		if !exist {
			connectedTrackerAddresses = append(connectedTrackerAddresses, AddrAndFilename{Addr: trackerAddress, Filename: filename})
		}
	}
	return nil
}

func ConnectToTracker(trackerAddress string, peerAddress string, filename string) error {
	conn, err := net.Dial("tcp", trackerAddress)
	if err != nil {
		return fmt.Errorf("connection failed: %v", err)
	}
	defer conn.Close()

	fmt.Printf("Connected to tracker %s for file %s\n", trackerAddress, filename)

	// Tạo message để gửi
	// Format: START:{peerAddress}:{fileName}
	message := fmt.Sprintf("START:%s:%s", peerAddress, filename)

	// Gửi message đến tracker
	if _, err := conn.Write([]byte(message)); err != nil {
		return fmt.Errorf("failed to send data: %v", err)
	}
	return nil
}

func GetListOfPeersForAFile(trackerAddress string, filename string) error {
	conn, err := net.Dial("tcp", trackerAddress)
	if err != nil {
		return fmt.Errorf("connection failed: %v", err)
	}
	defer conn.Close()

	// Tạo message để gửi
	// Format: LIST:{fileName}
	message := fmt.Sprintf("LIST:%s", filename)

	// Gửi message đến tracker
	if _, err := conn.Write([]byte(message)); err != nil {
		return fmt.Errorf("failed to send data: %v", err)
	}

	// Đọc phản hồi từ tracker
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('!')
	response = response[:len(response)-1] // Remove the last character (!)
	if err != nil {
		return fmt.Errorf("failed to read tracker response: %v", err)
	}

	fmt.Printf("Tracker response: %s\n", response)
	return nil
}

func DisconnectToTracker(peerAddress string) error {
	for _, tracker := range connectedTrackerAddresses {
		conn, err := net.Dial("tcp", tracker.Addr)
		if err != nil {
			return fmt.Errorf("connection failed: %v", err)
		}
		defer conn.Close()

		// Tạo message để gửi
		// Format: STOP:{peerAddress}
		message := fmt.Sprintf("STOP:%s", peerAddress)

		// Gửi message đến tracker
		if _, err := conn.Write([]byte(message)); err != nil {
			return fmt.Errorf("failed to send data: %v", err)
		}
	}
	return nil
}
func GetListOfTrackers() []AddrAndFilename {
	return connectedTrackerAddresses
}
