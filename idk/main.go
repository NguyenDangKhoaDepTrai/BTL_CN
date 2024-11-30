package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"tcp-app/client"
	"tcp-app/server"
	"tcp-app/torrent"
)

func main() {
	go func() {
		err := server.StartServer(":8080")
		if err != nil {
			log.Fatalf("Failed to start server: %v\n", err)
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Torrent Simulation App")
	fmt.Println("Commands:")
	fmt.Println("  download [torrent-file]  - Start downloading a torrent file")
	fmt.Println("  test [ip:port]          - Test connection to a peer")
	fmt.Println("  exit                     - Exit the program")
	fmt.Println("  clear                    - Clear the terminal")
	fmt.Println("  create [file]            - Create a torrent file from a source file")
	fmt.Println("  open [torrent-file]      - Open and display torrent file contents")
	fmt.Println("  test-file [filename]     - Test split and merge functionality")
	for {
		fmt.Print("> ") // CLI prompt
		commandLine, _ := reader.ReadString('\n')
		commandLine = strings.TrimSpace(commandLine)

		// Handle commands
		switch {
		case strings.HasPrefix(commandLine, "seed"):
			args := strings.Split(commandLine, " ")
			if len(args) < 2 {
				fmt.Println("Usage: seed [torrent-file]")
				continue
			}
			torrentFile := args[1]
			server.StartServer(torrentFile)
		case strings.HasPrefix(commandLine, "download"):
			args := strings.Split(commandLine, " ")
			if len(args) < 2 {
				fmt.Println("Usage: download [torrent-file]")
				continue
			}
			torrentFile := args[1]
			client.StartDownload(torrentFile)

		case strings.HasPrefix(commandLine, "test"):
			args := strings.Split(commandLine, " ")
			if len(args) < 2 {
				fmt.Println("Usage: test [ip:port]")
				continue
			}
			peerAddress := args[1]
			if err := client.TestConnection(peerAddress); err != nil {
				fmt.Printf("Connection failed: %v\n", err)
			} else {
				fmt.Printf("Successfully connected to %s\n", peerAddress)
			}

		case commandLine == "exit":
			fmt.Println("Exiting...")
			return

		case commandLine == "clear":
			fmt.Println("\033[H\033[2J") // Clear the terminal

		case strings.HasPrefix(commandLine, "create"):
			args := strings.Split(commandLine, " ")
			if len(args) < 2 {
				fmt.Println("Usage: create [file]")
				continue
			}
			sourceFile := args[1]
			torrentFileName, err := torrent.Create(sourceFile)
			if err != nil {
				fmt.Printf("Failed to create torrent file: %v\n", err)
			} else {
				fmt.Printf("Torrent file created successfully: %s\n", torrentFileName)
			}
		case strings.HasPrefix(commandLine, "open"):
			args := strings.Split(commandLine, " ")
			if len(args) < 2 {
				fmt.Println("Usage: open [torrent-file]")
				continue
			}
			torrentFile := args[1]
			torrent.Open(torrentFile)
		case strings.HasPrefix(commandLine, "check-file"):
			args := strings.Split(commandLine, " ")
			if len(args) < 2 {
				fmt.Println("Usage: check-file <filename>")
				return
			}
			filename := args[1]
			if err := torrent.TestSplitAndMerge(filename); err != nil {
				fmt.Printf("Test failed: %v\n", err)
				return
			}
		default:
			fmt.Println("Unknown command. Try again.")
		}
	}
}
