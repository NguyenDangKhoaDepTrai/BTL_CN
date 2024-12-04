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

// func getLocalIP() string {
// 	interfaces, err := net.Interfaces()
// 	if err != nil {
// 		return "unknown"
// 	}

// 	for _, iface := range interfaces {
// 		// Check if it's a wireless interface (common naming patterns)
// 		if strings.Contains(strings.ToLower(iface.Name), "wi-fi") ||
// 			strings.Contains(strings.ToLower(iface.Name), "wlan") {

// 			addrs, err := iface.Addrs()
// 			if err != nil {
// 				continue
// 			}

// 			for _, addr := range addrs {
// 				if ipnet, ok := addr.(*net.IPNet); ok {
// 					if ip4 := ipnet.IP.To4(); ip4 != nil {
// 						return ip4.String()
// 					}
// 				}
// 			}
// 		}
// 	}
// 	return "unknown"
// }

func main() {
	trackerAddress := "192.168.101.99:8080"
	peerAddress := "192.168.101.11"
	//peerAddress := "192.168.101.98"
	go func() {
		serverAddress := fmt.Sprintf("%s:8080", peerAddress)
		err := server.StartServer(serverAddress, trackerAddress)
		if err != nil {
			log.Fatalf("Failed to start server: %v\n", err)
		}
	}()
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n> ") // CLI prompt
		commandLine, _ := reader.ReadString('\n')
		commandLine = strings.TrimSpace(commandLine)

		// Handle commands
		switch {
		//-----------------------------------------------------------------------------------------------------
		case strings.HasPrefix(commandLine, "menu"):
			fmt.Println("Torrent Simulation App")
			fmt.Println("Commands:")
			fmt.Println("  connecttotracker [filename]       	- Connect to tracker and announce file")
			fmt.Println("  download [torrent-file]  		- Start downloading a file from a torrent file")
			fmt.Println("  test [ip:port]           		- Test connection to another peer")
			fmt.Println("  create [file]           		- Create a torrent file from a source file")
			fmt.Println("  open [torrent-file]      		- Open and display torrent file contents")
			fmt.Println("  test-file [filename]     		- Test split and merge functionality")
			fmt.Println("  clear                   		- Clear the terminal")
			fmt.Println("  exit                    		- Exit the program")
			continue
			//-----------------------------------------------------------------------------------------------------
			// case strings.HasPrefix(commandLine, "seed"):
			// 	args := strings.Split(commandLine, " ")
			// 	if len(args) < 2 {
			// 		fmt.Println("Usage: seed [torrent-file]")
			// 		continue
			// 	}
			// 	torrentFile := args[1]
			// 	server.StartServer(torrentFile)
		//-----------------------------------------------------------------------------------------------------
		case strings.HasPrefix(commandLine, "connecttotracker"):
			args := strings.Split(commandLine, " ")
			if len(args) != 2 {
				fmt.Println("Usage: connecttotracker [filename]")
				continue
			}
			filename := args[1]
			client.ConnectToTracker(trackerAddress, peerAddress, filename)
		//-----------------------------------------------------------------------------------------------------
		case strings.HasPrefix(commandLine, "create"):
			args := strings.Split(commandLine, " ")
			if len(args) < 2 {
				fmt.Println("Usage: create [file]")
				continue
			}
			sourceFile := args[1]
			torrentFileName, err := torrent.Create(sourceFile, trackerAddress)
			if err != nil {
				fmt.Printf("Failed to create torrent file: %v\n", err)
			} else {
				fmt.Printf("Torrent file created successfully: %s\n", torrentFileName)
			}
		//-----------------------------------------------------------------------------------------------------
		case strings.HasPrefix(commandLine, "download"):
			args := strings.Split(commandLine, " ")
			if len(args) < 2 {
				fmt.Println("Usage: download [torrent-file]")
				continue
			}
			torrentFile := args[1]
			client.StartDownload(torrentFile)
		//-----------------------------------------------------------------------------------------------------
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
		//-----------------------------------------------------------------------------------------------------
		case strings.HasPrefix(commandLine, "open"):
			args := strings.Split(commandLine, " ")
			if len(args) < 2 {
				fmt.Println("Usage: open [torrent-file]")
				continue
			}
			torrentFile := args[1]
			torrent.Open(torrentFile)
		//-----------------------------------------------------------------------------------------------------
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
		//-----------------------------------------------------------------------------------------------------
		case commandLine == "exit":
			fmt.Println("Exiting...")
			client.DisconnectToTracker(trackerAddress, peerAddress)
			return
		//-----------------------------------------------------------------------------------------------------
		case commandLine == "clear":
			fmt.Println("\033[H\033[2J") // Clear the terminal
		//-----------------------------------------------------------------------------------------------------
		default:
			fmt.Println("Unknown command. Try again.")
		}
	}
}
