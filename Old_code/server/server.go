package main

import (
	"fmt"
	"net"
	"os"
	"sync" // Import sync for safe concurrent access
)

// Declare a global slice to store connected IPs and a mutex for safe access
var connectedIPs []string
var mu sync.Mutex // Mutex to protect access to connectedIPs

// Function to handle each client connection
func handleConnection(conn net.Conn) {
	defer conn.Close()

	clientAddr := conn.RemoteAddr().String()
	fmt.Printf("Connection received from %s\n", clientAddr)

	// Lock the mutex before modifying the connectedIPs slice
	mu.Lock()
	connectedIPs = append(connectedIPs, clientAddr) // Append the client's IP to the list
	mu.Unlock()

	// Read data from the client
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading:", err.Error())
		return
	}
	data := string(buffer[:n])
	fmt.Printf("Received: %s\n", data)
}

func main() {
	// Define the host and port
	host := getDefaultInterfaceIP()
	port := "22236"
	address := net.JoinHostPort(host, port)
	fmt.Printf("Server listening on: %s\n", address)

	// Start the server and listen on the address
	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("Error starting server:", err.Error())
		os.Exit(1)
	}
	defer listener.Close()

	// Accept incoming connections and handle each in a goroutine
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err.Error())
			continue
		}
		go handleConnection(conn)
	}
}

// Function to get the default interface IP
func getDefaultInterfaceIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
