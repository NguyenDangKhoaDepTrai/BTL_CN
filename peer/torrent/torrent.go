package torrent

import (
	"crypto/sha1"
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
	Announce string        `bencode:"announce"`
	Info     []bencodeInfo `bencode:"info"`
}

// Open parses a torrent file
func Open(path string) ([]TorrentFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return []TorrentFile{}, err
	}
	defer file.Close()

	bto := bencodeTorrent{}
	err = bencode.Unmarshal(file, &bto)
	if err != nil {
		return []TorrentFile{}, err
	}

	torrentFiles, err := bto.toTorrentFile()
	if err != nil {
		return []TorrentFile{}, err
	}
	// fmt.Printf("bto.toTorrentFile()[0].Announce: %v\n", torrentFiles[0].Announce)
	// fmt.Printf("bto.toTorrentFile()[0].InfoHash: %x\n", torrentFiles[0].InfoHash)
	// fmt.Printf("bto.toTorrentFile()[0].PieceHashes: %v\n", torrentFiles[0].PieceHashes)
	// fmt.Printf("bto.toTorrentFile()[0].PieceLength: %d\n", torrentFiles[0].PieceLength)
	// fmt.Printf("bto.toTorrentFile()[0].Length: %d\n", torrentFiles[0].Length)
	// fmt.Printf("bto.toTorrentFile()[0].Name: %s\n", torrentFiles[0].Name)
	return torrentFiles, nil
}

// func (i bencodeInfo) hash() ([20]byte, error) {
// 	var buf bytes.Buffer
// 	err := bencode.Marshal(&buf, i)
// 	if err != nil {
// 		return [20]byte{}, err
// 	}
// 	h := sha1.Sum(buf.Bytes())
// 	return h, nil
// }

func (i *bencodeInfo) splitPieceHashes() ([][20]byte, error) {
	hashLen := 20 // Length of SHA-1 hash
	buf := []byte(i.Pieces)
	if len(buf)%hashLen != 0 {
		err := fmt.Errorf("received malformed pieces of length %d", len(buf))
		return nil, err
	}
	numHashes := len(buf) / hashLen
	hashes := make([][20]byte, numHashes)

	for i := 0; i < numHashes; i++ {
		copy(hashes[i][:], buf[i*hashLen:(i+1)*hashLen])
	}
	return hashes, nil
}

func (bto *bencodeTorrent) toTorrentFile() ([]TorrentFile, error) {
	torrentFiles := []TorrentFile{}
	for _, info := range bto.Info {
		// infoHash, err := info.hash()
		// if err != nil {
		// 	return []TorrentFile{}, err
		// }
		infoHash := sha1.Sum([]byte(info.Name))
		pieceHashes, err := info.splitPieceHashes()
		if err != nil {
			return []TorrentFile{}, err
		}
		t := TorrentFile{
			Announce:    bto.Announce,
			InfoHash:    infoHash,
			PieceHashes: pieceHashes,
			PieceLength: info.PieceLength,
			Length:      info.Length,
			Name:        info.Name,
		}
		torrentFiles = append(torrentFiles, t)
	}
	return torrentFiles, nil
}

// []TorrentFile to bencodeTorrent
func toBencodeTorrent(t []TorrentFile) (bencodeTorrent, error) {
	bto := bencodeTorrent{
		Announce: t[0].Announce,
	}
	for _, torrentFile := range t {
		bto.Info = append(bto.Info, torrentFile.toBencodeInfo())
	}
	return bto, nil
}

func (t *TorrentFile) toBencodeInfo() bencodeInfo {
	// Concatenate all piece hashes into a single byte slice
	var pieces []byte
	for _, hash := range t.PieceHashes {
		pieces = append(pieces, hash[:]...)
	}

	return bencodeInfo{
		Pieces:      string(pieces),
		PieceLength: t.PieceLength,
		Length:      t.Length,
		Name:        t.Name,
	}
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
func CreateTorrent(paths []string, trackerURL string) ([]TorrentFile, error) {
	torrentFiles := []TorrentFile{}
	for _, path := range paths {
		pieceLength := 256 * 1024 // 256 KB
		file, err := os.Open("files/" + path)
		if err != nil {
			return []TorrentFile{}, err
		}
		defer file.Close()

		fileInfo, err := file.Stat()
		if err != nil {
			return nil, err
		}
		// Hash the file stat
		infoHash := sha1.Sum([]byte(fileInfo.Name()))

		// Use the new function to split the file into pieces
		pieces, err := splitFileIntoPieces(file, pieceLength)
		if err != nil {
			return nil, err
		}

		// Calculate pieces hashes
		var piecesHashes [][20]byte
		for _, piece := range pieces {
			hash := sha1.Sum(piece)
			piecesHashes = append(piecesHashes, hash)
		}

		// Create torrent file from the data above
		torrentFile := TorrentFile{
			Announce:    trackerURL,
			InfoHash:    infoHash,
			PieceHashes: piecesHashes,
			PieceLength: pieceLength,
			Length:      int(fileInfo.Size()),
			Name:        fileInfo.Name(),
		}

		torrentFiles = append(torrentFiles, torrentFile)
	}

	return torrentFiles, nil
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
func (t bencodeTorrent) createTorrentFile(path string) error {
	//fmt.Println("Creating torrent file:", t)
	file, err := os.Create("torrent_files/" + path)
	if err != nil {
		return err
	}
	defer file.Close()

	err = bencode.Marshal(file, t)
	if err != nil {
		return err
	}

	return nil
}

func Create(path []string, trackerURL string) (torrentPath string, err error) {
	torrentFiles, err := CreateTorrent(path, trackerURL)
	if err != nil {
		return "", err
	}
	// Generate torrent file name from paths by the hash of the combined paths
	combinedPath := strings.Join(path, ",")
	//fmt.Printf("combinedPath: %s\n", combinedPath)
	hash_file_name := sha1.Sum([]byte(combinedPath))
	torrentFileName := fmt.Sprintf("%x.torrent", hash_file_name)
	// convert torrentFiles to bencodeTorrent
	bto, err := toBencodeTorrent(torrentFiles)
	if err != nil {
		return "", err
	}
	err = bto.createTorrentFile(torrentFileName)
	if err != nil {
		return "", err
	}

	// Create torrent_info.json
	// torrentInfo := map[string]string{
	// 	"InfoHash": torrentFileName,
	// }
	// jsonData, err := json.Marshal(torrentInfo)
	// if err != nil {
	// 	return "", err
	// }
	// err = os.WriteFile("torrent_info.json", jsonData, 0644) // insert jsonData into torrent_info.json
	// if err != nil {
	// 	return "", err
	// }
	// files, err := ListTorrentFiles()
	// if err != nil {
	// 	return "", err
	// }
	// for _, file := range files {
	// 	fmt.Println(file)
	// }
	return torrentFileName, nil
}

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

// MergePieces combines pieces into a single file
func (t *TorrentFile) MergePieces(outputPath string, pieces map[int][]byte) error {
	file, err := os.Create("files/" + outputPath)
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
		if _, err := file.Write(data); err != nil {
			return fmt.Errorf("failed to write piece %d: %v", i, err)
		}
	}
	return nil
}
