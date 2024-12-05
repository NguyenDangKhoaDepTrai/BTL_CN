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

	"github.com/jackpal/bencode-go" // or another bencode library
)

// func getLocalIP() string { //this function is only used for windows
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

//				for _, addr := range addrs {
//					if ipnet, ok := addr.(*net.IPNet); ok {
//						if ip4 := ipnet.IP.To4(); ip4 != nil {
//							return ip4.String()
//						}
//					}
//				}
//			}
//		}
//		return "unknown"
//	}
func getTrackerAddress(torrentPath string) (string, error) {
	file, err := os.Open(torrentPath)
	if err != nil {
		return "", fmt.Errorf("error opening torrent file: %v", err)
	}
	defer file.Close()

	// Define a struct to decode the torrent file
	type TorrentFile struct {
		Announce string `bencode:"announce"`
	}

	var torrent TorrentFile
	err = bencode.Unmarshal(file, &torrent)
	if err != nil {
		return "", fmt.Errorf("error decoding torrent file: %v", err)
	}

	return torrent.Announce, nil
}

func removeFromSlice(slice []AddrAndFilename, item AddrAndFilename) []AddrAndFilename {
	for i, v := range slice {
		if v == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

type AddrAndFilename struct {
	Addr     string
	Filename string
}

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
	connectedTrackerAddresses := []AddrAndFilename{}
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
			fmt.Println("  disconnecttotracker [filename]    	- Disconnect from tracker")
			fmt.Println("  getlistofpeers [filename] 	- Get list of peers for a specific file")
			fmt.Println("  getlistoftrackers 		- Get list of trackers connected")
			fmt.Println("  download [torrent-file]  		- Start downloading a file from a torrent file")
			fmt.Println("  test [ip:port]           		- Test connection to another peer")
			fmt.Println("  create [file] [tracker-address]           		- Create a torrent file from a source file")
			//fmt.Println("  open [torrent-file]      		- Open and display torrent file contents")
			//fmt.Println("  test-file [filename]     		- Test split and merge functionality")
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
			trackerAddress, err := getTrackerAddress(filename + ".torrent")
			if err != nil {
				fmt.Printf("Failed to get tracker address, please check the torrent file: %v\n", err)
				continue
			}
			err = client.ConnectToTracker(trackerAddress, peerAddress, filename)
			if err != nil {
				fmt.Printf("Failed to connect to tracker: %v\n", err)
				return
			}
			exist := false
			for _, tracker := range connectedTrackerAddresses {
				if tracker.Addr == trackerAddress && tracker.Filename == filename {
					exist = true
					break
				}
			}
			if !exist {
				connectedTrackerAddresses = append(connectedTrackerAddresses, AddrAndFilename{Addr: trackerAddress, Filename: filename})
			}
		//-----------------------------------------------------------------------------------------------------
		case strings.HasPrefix(commandLine, "disconnecttotracker"):
			args := strings.Split(commandLine, " ")
			if len(args) != 2 {
				fmt.Println("Usage: disconnecttotracker [filename]")
				continue
			}
			filename := args[1]
			trackerAddress, err := getTrackerAddress(filename + ".torrent")
			if err != nil {
				fmt.Printf("Failed to get tracker address, please check the torrent file: %v\n", err)
				continue
			}
			err = client.DisconnectToTrackerForAFile(trackerAddress, peerAddress, filename)
			if err != nil {
				fmt.Printf("Failed to disconnect to tracker: %v\n", err)
				continue
			}
			connectedTrackerAddresses = removeFromSlice(connectedTrackerAddresses, AddrAndFilename{Addr: trackerAddress, Filename: filename})
		//-----------------------------------------------------------------------------------------------------
		case strings.HasPrefix(commandLine, "getlistofpeers"):
			args := strings.Split(commandLine, " ")
			if len(args) != 2 {
				fmt.Println("Usage: getlistofpeers [filename]")
				continue
			}
			filename := args[1]
			trackerAddress, err := getTrackerAddress(filename + ".torrent")
			if err != nil {
				fmt.Printf("Failed to get tracker address, please check the torrent file: %v\n", err)
				continue
			}
			found := false
			for _, tracker := range connectedTrackerAddresses {
				if tracker.Addr == trackerAddress && tracker.Filename == filename {
					found = true
					break
				}
			}
			if !found {
				fmt.Println("You are not connected to tracker for this file")
				continue
			}
			err = client.GetListOfPeersForAFile(trackerAddress, peerAddress, filename)
			if err != nil {
				fmt.Printf("Failed to get list of peers: %v\n", err)
				continue
			}
		//-----------------------------------------------------------------------------------------------------
		case strings.HasPrefix(commandLine, "getlistoftrackers"):
			if len(connectedTrackerAddresses) == 0 {
				fmt.Println("No trackers connected")
				continue
			}
			fmt.Println("List of trackers connected:")
			for _, trackerAddress := range connectedTrackerAddresses {
				fmt.Println(trackerAddress)
			}
		//-----------------------------------------------------------------------------------------------------
		case strings.HasPrefix(commandLine, "create"):
			args := strings.Split(commandLine, " ")
			if len(args) < 3 {
				fmt.Println("Usage: create [file] [tracker-address]")
				continue
			}
			sourceFile := args[1]
			trackerAddress := args[2]
			trackerAddress = trackerAddress + ":8080"
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
		// case strings.HasPrefix(commandLine, "open"):
		// 	args := strings.Split(commandLine, " ")
		// 	if len(args) < 2 {
		// 		fmt.Println("Usage: open [torrent-file]")
		// 		continue
		// 	}
		// 	torrentFile := args[1]
		// 	torrent.Open(torrentFile)
		//-----------------------------------------------------------------------------------------------------
		// case strings.HasPrefix(commandLine, "check-file"):
		// 	args := strings.Split(commandLine, " ")
		// 	if len(args) < 2 {
		// 		fmt.Println("Usage: check-file <filename>")
		// 		return
		// 	}
		// 	filename := args[1]
		// 	if err := torrent.TestSplitAndMerge(filename); err != nil {
		// 		fmt.Printf("Test failed: %v\n", err)
		// 		return
		// 	}
		//-----------------------------------------------------------------------------------------------------
		case commandLine == "exit":
			fmt.Println("Exiting...")
			for _, trackerAddress := range connectedTrackerAddresses {
				client.DisconnectToTracker(trackerAddress.Addr, peerAddress)
			}
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
