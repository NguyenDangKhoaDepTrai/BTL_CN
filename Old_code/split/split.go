package main

import (
	"fmt"
	"io"
	"os"
)

const pieceSize = 512 * 1024 // 512KB

func splitFile(filename string) ([]string, error) {
	var pieceFiles []string
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buffer := make([]byte, pieceSize)
	count := 0

	for {
		bytesRead, err := file.Read(buffer)
		if bytesRead == 0 {
			break
		}

		pieceFilename := fmt.Sprintf("piece_%d.dat", count)
		pieceFiles = append(pieceFiles, pieceFilename)

		pieceFile, err := os.Create(pieceFilename)
		if err != nil {
			return nil, err
		}

		_, err = pieceFile.Write(buffer[:bytesRead])
		if err != nil {
			pieceFile.Close()
			return nil, err
		}
		pieceFile.Close()
		count++

		if err == io.EOF {
			break
		}
	}

	return pieceFiles, nil
}

func main() {
	var filename string
	fmt.Print("Enter the filename: ")
	fmt.Scan(&filename)
	pieces, err := splitFile(filename)
	if err != nil {
		fmt.Println("Error splitting file:", err)
		return
	}
	fmt.Println("Split file into pieces:", pieces)
}
