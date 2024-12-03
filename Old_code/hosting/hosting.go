package main

import (
	"fmt"
	"net"
	"os"
	"sync"
)

const pieceSize = 512 * 1024 // 512KB

func sendFile(pieceFilename string, conn net.Conn) {
	file, err := os.Open(pieceFilename)
	if err != nil {
		fmt.Println("Error opening file:", err)
		os.Exit(1)
		return
	}
	defer file.Close()

	buffer := make([]byte, pieceSize)
	for {
		bytesRead, err := file.Read(buffer)
		if err != nil {
			break
		}
		conn.Write(buffer[:bytesRead])
	}

	fmt.Printf("Sent %s on port %s\n", pieceFilename, conn.LocalAddr())
}

func startServer(port int, filename string, wg *sync.WaitGroup) {
	defer wg.Done()

	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address) // Tạo socket server để lắng nghe trên cổng port
	if err != nil {
		fmt.Printf("Error starting server on port %d: %v\n", port, err)
		return
	}
	defer listener.Close()

	fmt.Printf("Server is listening on port %d for file %s\n", port, filename)

	for {
		conn, err := listener.Accept() // Chấp nhận kết nối từ client
		if err != nil {
			fmt.Printf("Connection error on port %d: %v\n", port, err)
			continue
		}
		go func(conn net.Conn) {
			defer conn.Close()
			sendFile(filename, conn)
		}(conn)
	}
}

func main() {
	// Define port-to-file mapping
	var numPieces int
	fmt.Print("Enter the number of pieces to serve: ")
	fmt.Scan(&numPieces)
	portFileMap := make(map[int]string, numPieces)
	for i := 0; i < numPieces; i++ {
		portFileMap[8080+i] = fmt.Sprintf("piece_%d.dat", i)
	}

	var wg sync.WaitGroup

	// Start a server for each port
	for port, filename := range portFileMap {
		wg.Add(1)
		go startServer(port, filename, &wg)
	}

	// Wait for all servers (this will block forever since servers run continuously)
	wg.Wait()
}
