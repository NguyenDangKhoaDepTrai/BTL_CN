package torrent

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jackpal/bencode-go"
)

// TorrentFile encodes the metadata from a .torrent file
type TorrentFile struct {
	Announce    string
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
}

type bencodeInfo struct {
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
}

type bencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     bencodeInfo `bencode:"info"`
}

// Open parses a torrent file
func Open(path string) (TorrentFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return TorrentFile{}, err
	}
	defer file.Close()

	bto := bencodeTorrent{}
	err = bencode.Unmarshal(file, &bto)
	if err != nil {
		return TorrentFile{}, err
	}
	fmt.Println(bto.toTorrentFile())
	return bto.toTorrentFile()
}

func (i *bencodeInfo) hash() ([20]byte, error) {
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, *i)
	if err != nil {
		return [20]byte{}, err
	}
	h := sha1.Sum(buf.Bytes())
	return h, nil
}

func (i *bencodeInfo) splitPieceHashes() ([][20]byte, error) {
	hashLen := 20 // Length of SHA-1 hash
	buf := []byte(i.Pieces)
	if len(buf)%hashLen != 0 {
		err := fmt.Errorf("Received malformed pieces of length %d", len(buf))
		return nil, err
	}
	numHashes := len(buf) / hashLen
	hashes := make([][20]byte, numHashes)

	for i := 0; i < numHashes; i++ {
		copy(hashes[i][:], buf[i*hashLen:(i+1)*hashLen])
	}
	return hashes, nil
}

func (bto *bencodeTorrent) toTorrentFile() (TorrentFile, error) {
	infoHash, err := bto.Info.hash()
	if err != nil {
		return TorrentFile{}, err
	}
	pieceHashes, err := bto.Info.splitPieceHashes()
	if err != nil {
		return TorrentFile{}, err
	}
	t := TorrentFile{
		Announce:    bto.Announce,
		InfoHash:    infoHash,
		PieceHashes: pieceHashes,
		PieceLength: bto.Info.PieceLength,
		Length:      bto.Info.Length,
		Name:        bto.Info.Name,
	}
	return t, nil
}

// splitFileIntoPieces reads a file and splits it into pieces of the given length.
func splitFileIntoPieces(file *os.File, pieceLength int) ([][]byte, error) {
	var pieces [][]byte
	buf := make([]byte, pieceLength)
	for {
		n, err := file.Read(buf)
		if n == 0 {
			break
		}
		if err != nil && err != io.EOF {
			return nil, err
		}
		piece := make([]byte, n)
		copy(piece, buf[:n])
		pieces = append(pieces, piece)
	}
	return pieces, nil
}

// CreateTorrent builds a TorrentFile from a file path and tracker URL
func CreateTorrent(path string, trackerURL string) (TorrentFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return TorrentFile{}, err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return TorrentFile{}, err
	}

	// Create bencode structs
	bto := bencodeTorrent{
		Announce: trackerURL,
		Info: bencodeInfo{
			PieceLength: 262144, // Standard piece length of 256KB
			Name:        fileInfo.Name(),
			Length:      int(fileInfo.Size()),
		},
	}

	// Use the new function to split the file into pieces
	pieces, err := splitFileIntoPieces(file, bto.Info.PieceLength)
	if err != nil {
		return TorrentFile{}, err
	}

	// Calculate pieces hashes
	var piecesHashes []byte
	for _, piece := range pieces {
		hash := sha1.Sum(piece)
		piecesHashes = append(piecesHashes, hash[:]...)
	}
	bto.Info.Pieces = string(piecesHashes)

	return bto.toTorrentFile()
}

// StreamFilePieces streams file pieces to a client without hashing
func StreamFilePieces(filePath string, pieceLength int) ([][]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Use the same function to split the file into pieces
	return splitFileIntoPieces(file, pieceLength)
}

// Create saves a TorrentFile as a .torrent file
func (t *TorrentFile) createTorrentFile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	bto := bencodeTorrent{
		Announce: t.Announce,
		Info: bencodeInfo{
			Pieces: string(bytes.Join(func() [][]byte {
				pieces := make([][]byte, len(t.PieceHashes))
				for i := range t.PieceHashes {
					pieces[i] = t.PieceHashes[i][:]
				}
				return pieces
			}(), []byte{})),
			PieceLength: t.PieceLength,
			Length:      t.Length,
			Name:        t.Name,
		},
	}

	return bencode.Marshal(file, bto)
}

func Create(path string) (torrentPath string, err error) {
	trackerURL := "http://localhost:8080/announce"
	torrentFile, err := CreateTorrent(path, trackerURL)
	if err != nil {
		return "", err
	}
	torrentFileName := fmt.Sprintf("%s.torrent", path)
	err = torrentFile.createTorrentFile(torrentFileName)
	if err != nil {
		return "", err
	}

	torrentInfo := map[string]string{
		"FilePath": path,
		"FileName": torrentFile.Name,
		"InfoHash": fmt.Sprintf("%x", torrentFile.InfoHash),
		// Print the list of piece hashes
		"PieceHashes": fmt.Sprintf("%x", torrentFile.PieceHashes),
	}
	jsonData, err := json.Marshal(torrentInfo)
	if err != nil {
		return "", err
	}
	err = os.WriteFile("torrent_info.json", jsonData, 0644)
	if err != nil {
		return "", err
	}
	return torrentFileName, nil
}

func (t *TorrentFile) ReadPiece(index int) ([]byte, error) {
	// Validate piece index
	if index < 0 || index >= len(t.PieceHashes) {
		return nil, fmt.Errorf("invalid piece index %d", index)
	}

	// Open the underlying file
	file, err := os.Open(t.Name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Calculate piece size and offset
	pieceOffset := int64(index * t.PieceLength)
	pieceSize := t.PieceLength

	// Handle last piece, which might be smaller
	if index == len(t.PieceHashes)-1 {
		lastPieceSize := t.Length - (index * t.PieceLength)
		if lastPieceSize < pieceSize {
			pieceSize = lastPieceSize
		}
	}

	// Seek to the piece location
	_, err = file.Seek(pieceOffset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	// Read the piece
	piece := make([]byte, pieceSize)
	n, err := io.ReadFull(file, piece)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}

	// Verify piece hash
	hash := sha1.Sum(piece[:n])
	if !bytes.Equal(hash[:], t.PieceHashes[index][:]) {
		return nil, fmt.Errorf("piece %d failed hash verification", index)
	}

	return piece[:n], nil
}

// MergePieces combines pieces into a single file
func (t *TorrentFile) MergePieces(outputPath string, pieces map[int]string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// Write pieces in order
	for i := 0; i < len(t.PieceHashes); i++ {
		data, exists := pieces[i]
		if !exists {
			return fmt.Errorf("missing piece %d", i)
		}
		if _, err := file.WriteString(data); err != nil {
			return fmt.Errorf("failed to write piece %d: %v", i, err)
		}
	}

	return nil
}

// TestSplitAndMerge tests the split and merge functionality
func TestSplitAndMerge(filepath string) error {
	// Create a temporary torrent file structure
	t := &TorrentFile{
		PieceLength: 256 * 1024, // 256KB pieces
		Name:        filepath,
	}

	// Use StreamFilePieces to split the file
	pieceBytes, err := StreamFilePieces(filepath, t.PieceLength)
	if err != nil {
		return fmt.Errorf("failed to split file: %v", err)
	}

	// Convert pieces to map and calculate hashes
	pieces := make(map[int]string)
	for i, piece := range pieceBytes {
		pieces[i] = string(piece)
		pieceHash := sha1.Sum(piece)
		t.PieceHashes = append(t.PieceHashes, pieceHash)
	}

	// Get extension by splitting on dots and taking the last part
	parts := strings.Split(filepath, ".")
	var ext string
	if len(parts) > 1 {
		ext = "." + parts[len(parts)-1]
	}
	baseName := strings.TrimSuffix(filepath, ext)
	outputPath := baseName + "-test" + ext

	// Merge pieces back
	if err := t.MergePieces(outputPath, pieces); err != nil {
		return fmt.Errorf("merge failed: %v", err)
	}

	fmt.Printf("Successfully split and merged file:\nOriginal: %s\nNew: %s\n", filepath, outputPath)
	return nil
}
