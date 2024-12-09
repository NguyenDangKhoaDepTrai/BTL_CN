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

// func removeFromSlice(slice []AddrAndFilename, item AddrAndFilename) []AddrAndFilename {
// 	for i, v := range slice {
// 		if v == item {
// 			return append(slice[:i], slice[i+1:]...)
// 		}
// 	}
// 	return slice
// }

func main() {
	var peerAddress string
	fmt.Print("Enter your peer address (e.g., 192.168.101.92): ")
	fmt.Scanln(&peerAddress)
	go func() {
		serverAddress := fmt.Sprintf("%s:8080", peerAddress)
		err := server.StartServer(serverAddress)
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
			fmt.Println("  getlistofpeers [one torrent-file] 							- Get list of peers for a specific torrent file")
			fmt.Println("  getlistoftrackers 											- Get list of trackers connected")
			fmt.Println("  download [torrent-file] [another-peer-address]  				- Start downloading a file from a torrent file")
			fmt.Println("  test [peer-address]           								- Test connection to another peer")
			fmt.Println("  create [tracker-address] [files]         					- Create a torrent file from multiple source files")
			fmt.Println("  clear                   										- Clear the terminal")
			fmt.Println("  exit                    										- Exit the program")
			continue
			//-----------------------------------------------------------------------------------------------------
		case strings.HasPrefix(commandLine, "announcetotracker"):
			args := strings.Split(commandLine, " ")
			if len(args) != 2 {
				fmt.Println("Usage: announcetotracker [only one torrent-file]")
				continue
			}
			torrentfilename := args[1]
			err := client.AnnounceToTracker(peerAddress, torrentfilename)
			if err != nil {
				fmt.Printf("Failed to announce to tracker: %v\n", err)
			}
		case strings.HasPrefix(commandLine, "getlistofpeers"):
			args := strings.Split(commandLine, " ")
			if len(args) != 2 {
				fmt.Println("Usage: getlistofpeers [one torrent-file]")
				continue
			}
			torrentfilename := args[1]
			tfs, err := torrent.Open("torrent_files/" + torrentfilename)
			if err != nil {
				fmt.Printf("Error opening torrent file: %v\n", err)
				continue
			}
			trackerAddress := ""
			filename := ""
			for _, tf := range tfs {
				trackerAddress = tf.Announce
				filename = tf.Name
				err := client.GetListOfPeersForAFile(trackerAddress, filename)
				if err != nil {
					fmt.Printf("Failed to get list of peers: %v\n", err)
					continue
				}
			}
		//-----------------------------------------------------------------------------------------------------
		case strings.HasPrefix(commandLine, "getlistoftrackers"):
			if len(client.GetListOfTrackers()) == 0 {
				fmt.Println("No trackers connected")
				continue
			}
			fmt.Println("List of trackers connected:")
			for _, trackerAddress := range client.GetListOfTrackers() {
				fmt.Println(trackerAddress)
			}
		//-----------------------------------------------------------------------------------------------------
		case strings.HasPrefix(commandLine, "create"):
			args := strings.Split(commandLine, " ")
			if len(args) <= 2 {
				fmt.Println("Usage: create [tracker-address] [files]")
				continue
			}
			trackerAddress := args[1]
			sourceFiles := args[2:]
			trackerAddress = trackerAddress + ":8081"
			torrentFileName, err := torrent.Create(sourceFiles, trackerAddress)
			if err != nil {
				fmt.Printf("Failed to create torrent file: %v\n", err)
			} else {
				fmt.Printf("Torrent file created successfully: %s\n", torrentFileName)
			}
		//-----------------------------------------------------------------------------------------------------
		case strings.HasPrefix(commandLine, "download"):
			args := strings.Split(commandLine, " ")
			if len(args) < 3 {
				fmt.Println("Usage: download [torrent-file] [another-peer-address]")
				continue
			}
			torrentFile := args[1]
			anotherPeerAddress := args[2:]
			client.StartDownload(torrentFile, anotherPeerAddress, peerAddress)
		//-----------------------------------------------------------------------------------------------------
		case strings.HasPrefix(commandLine, "test"):
			args := strings.Split(commandLine, " ")
			if len(args) < 2 {
				fmt.Println("Usage: test [ip]")
				continue
			}
			peerAddress := args[1]
			peerAddress = peerAddress + ":8080"
			if err := client.TestConnection(peerAddress); err != nil {
				fmt.Printf("Connection failed: %v\n", err)
			} else {
				fmt.Printf("Successfully connected to %s\n", peerAddress)
			}
		//-----------------------------------------------------------------------------------------------------
		case commandLine == "exit":
			fmt.Println("Exiting...")
			client.DisconnectToTracker(peerAddress)
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
