package tracker

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var peers []string

func NewConnection(address string) error {
	peers = append(peers, address)
	return nil
}

func GetPeers() []string {
	return peers
}
func main() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n> ") // CLI prompt
		commandLine, _ := reader.ReadString('\n')
		commandLine = strings.TrimSpace(commandLine)

		// Handle commands
		switch {
		case strings.HasPrefix(commandLine, "list"):
			fmt.Println(GetPeers())
		//-----------------------------------------------------------------------------------------------------
		case commandLine == "exit":
			fmt.Println("Exiting...")
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
