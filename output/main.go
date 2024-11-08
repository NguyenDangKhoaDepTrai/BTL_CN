package main

import (
	"fmt"
	"net"
	"os"
	"sync"
)

const pieceSize = 512 * 1024 // 512KB

func receiveFile(port int, filename string, wg *sync.WaitGroup) error {
	defer wg.Done()

	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port)) // Tạo socket client để kết nối với server trên localhost và cổng port
	if err != nil {
		return fmt.Errorf("connection error on port %d: %v", port, err)
	}
	defer conn.Close()

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file %s: %v", filename, err)
	}
	defer file.Close()

	buffer := make([]byte, pieceSize)
	for {
		bytesRead, err := conn.Read(buffer)
		if err != nil {
			break
		}
		_, writeErr := file.Write(buffer[:bytesRead])
		if writeErr != nil {
			return fmt.Errorf("error writing to file %s: %v", filename, writeErr)
		}
	}
	fmt.Printf("Received and saved: %s from port %d\n", filename, port)
	return nil
}

func mergePieces(numPieces int, outputFile string) error {
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer outFile.Close()

	buffer := make([]byte, pieceSize)
	for i := 0; i < numPieces; i++ {
		pieceFile := fmt.Sprintf("received_piece_%d.dat", i)

		piece, err := os.Open(pieceFile)
		if err != nil {
			return fmt.Errorf("error opening piece %d: %v", i, err)
		}

		for {
			bytesRead, err := piece.Read(buffer)
			if err != nil {
				break
			}
			_, writeErr := outFile.Write(buffer[:bytesRead])
			if writeErr != nil {
				piece.Close()
				return fmt.Errorf("error writing to output file: %v", writeErr)
			}
		}

		piece.Close()
		// Optionally remove the piece file after merging
		os.Remove(pieceFile)
	}

	fmt.Printf("Successfully merged %d pieces into %s\n", numPieces, outputFile)
	return nil
}

func main() {
	var numPieces int
	fmt.Print("Enter the number of pieces to receive: ")
	fmt.Scan(&numPieces)

	var wg sync.WaitGroup

	// Receive all pieces concurrently
	for i := 0; i < numPieces; i++ {
		wg.Add(1)
		port := 8080 + i
		filename := fmt.Sprintf("received_piece_%d.dat", i)

		go func(port int, filename string) {
			if err := receiveFile(port, filename, &wg); err != nil {
				fmt.Printf("Error receiving file on port %d: %v\n", port, err)
			}
		}(port, filename)
	}

	// Wait for all pieces to be received
	wg.Wait()

	// Merge the pieces
	var outputFile string
	fmt.Print("Enter the output file name: ")
	fmt.Scan(&outputFile)
	if err := mergePieces(numPieces, outputFile); err != nil {
		fmt.Printf("Error merging pieces: %v\n", err)
		return
	}

	fmt.Println("All pieces received and merged successfully!")
}
