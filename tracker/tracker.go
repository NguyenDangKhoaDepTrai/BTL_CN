package main

// Add to imports if not present
import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

var peerInfo = make(map[string][]string)

func AddPeer(peerAddr string, fileName string) error {
	peerInfo[fileName] = append(peerInfo[fileName], peerAddr)
	return nil
}

func removeFromSlice(slice []string, item string) []string {
	for i, v := range slice {
		if v == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

func RemovePeer(peerAddr string) error {
	for fileName, peers := range peerInfo {
		peerInfo[fileName] = removeFromSlice(peers, peerAddr)
	}
	return nil
}

func GetConnectedPeers() []string {
	var peers []string
	for _, peers := range peerInfo {
		peers = append(peers, peers...)
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
		err = AddPeer(peerAddr, fileName)
		if err != nil {
			fmt.Printf("Error adding peer: %v\n", err)
			return
		}
	case strings.HasPrefix(data, "STOP:"):
		err = RemovePeer(peerAddr)
		if err != nil {
			fmt.Printf("Error removing peer: %v\n", err)
			return
		}
	}

	// print peerInfo
	fmt.Println(peerInfo)

	// Send response to peer
	response, err := json.Marshal(peerInfo)
	if err != nil {
		fmt.Printf("Error marshaling response: %v\n", err)
		return
	}
	_, err = conn.Write(append(response, '!')) // Adding '!' as message delimiter
	fmt.Printf("Sent response to peer: %s\n----------------------------------------------- \n", string(response))
	if err != nil {
		fmt.Printf("Error sending response to peer: %v\n", err)
		return
	}
}

func main() {
	trackerAddress := "192.168.101.11:8080"
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
