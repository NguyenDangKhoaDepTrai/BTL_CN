package main

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"time"
)

// Function to send client information to the server
func sendInfo(conn net.Conn, clientHost string, clientPort int) {
	info := fmt.Sprintf("%s:%d", clientHost, clientPort)
	_, err := conn.Write([]byte(info))
	if err != nil {
		fmt.Println("Error sending data:", err.Error())
		return
	}
}

// Function to establish a new connection
func newConnection(tid int, host string, port int) {
	address := net.JoinHostPort(host, strconv.Itoa(port))
	fmt.Printf("Thread ID %d connecting to %s\n", tid, address)

	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Println("Connection error:", err.Error())
		return
	}
	defer conn.Close()

	// Get client's actual IP and port
	clientHost, clientPort := getDefaultInterfaceIP()

	// Send IP and port info to the server
	sendInfo(conn, clientHost, clientPort)

	// Simulate a delay with a loop
	for i := 0; i < 3; i++ {
		fmt.Printf("Let me, ID=%d sleep in %ds\n", tid, 3-i)
		time.Sleep(1 * time.Second)
	}
	fmt.Printf("OK! I am ID=%d done here\n", tid)
}

// Function to connect multiple clients in parallel
func connectServer(threadNum int, host string, port int) {
	for i := 0; i < threadNum; i++ {
		go newConnection(i, host, port)
	}
	// Allow time for goroutines to finish
	time.Sleep(time.Duration(threadNum) * time.Second)
}

// Helper function to get the default interface IP
func getDefaultInterfaceIP() (string, int) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1", 0
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), localAddr.Port
}

func main() {
	// Define command-line arguments for server IP, port, and client number
	serverIP := flag.String("server-ip", "192.168.0.108", "Server IP address")
	serverPort := flag.Int("server-port", 22236, "Server port number")
	clientNum := flag.Int("client-num", 1, "Number of clients to connect")
	flag.Parse()

	connectServer(*clientNum, *serverIP, *serverPort)
}
