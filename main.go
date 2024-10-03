package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

const MAX_MSG_SIZE = 1024

var (
	consoleMutex sync.Mutex
)

// handleConnection handles incoming messages from the peer.
func handleConnection(conn net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()

	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			consolePrint("Peer disconnected.\n")
			return
		}
		// Print the received message and overwrite "You: "
		consolePrint("\rPeer: " + message) // Overwrite the line with "Peer: "
		consolePrint("You: ")              // Reprint the prompt
	}
}

// sendMessage sends messages from the local user to the peer.
func sendMessage(conn net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()

	reader := bufio.NewReader(os.Stdin)
	for {
		// Prompt before reading input
		consolePrint("You: ")

		msg, _ := reader.ReadString('\n')
		msg = strings.TrimSpace(msg)
		if len(msg) > MAX_MSG_SIZE {
			consolePrint("Message too long. Max length is 1024 characters.\n")
			continue
		}
		if _, err := conn.Write([]byte(msg + "\n")); err != nil {
			consolePrint("Failed to send message.\n")
			return
		}
	}
}

// consolePrint prints messages to the console in a thread-safe manner.
func consolePrint(message string) {
	consoleMutex.Lock()
	defer consoleMutex.Unlock()
	fmt.Print(message)
}

// handleInterrupt listens for Ctrl+C and gracefully closes the connection.
func handleInterrupt(conn net.Conn) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	consolePrint("\nDisconnecting...\n")
	conn.Close()
	os.Exit(0)
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run chat.go [local|remote] [IP:port]")
		os.Exit(1)
	}

	mode := os.Args[1]
	address := os.Args[2]

	var conn net.Conn
	var err error

	if mode == "local" {
		listener, err := net.Listen("tcp", address)
		if err != nil {
			fmt.Println("Error starting listener:", err)
			os.Exit(1)
		}
		defer listener.Close()

		consolePrint("Waiting for a peer to connect...\n")
		conn, err = listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			os.Exit(1)
		}
		consolePrint("Peer connected.\n")
	} else if mode == "remote" {
		conn, err = net.Dial("tcp", address)
		if err != nil {
			fmt.Println("Error connecting to peer:", err)
			os.Exit(1)
		}
		consolePrint("Connected to peer.\n")
	} else {
		fmt.Println("Invalid mode. Use 'local' to listen or 'remote' to connect.")
		os.Exit(1)
	}
	defer conn.Close()

	wg := &sync.WaitGroup{}
	wg.Add(2)

	// Start goroutines for handling incoming and outgoing messages
	go handleConnection(conn, wg)
	go sendMessage(conn, wg)

	// Handle Ctrl+C interrupt
	go handleInterrupt(conn)

	// Wait for goroutines to finish
	wg.Wait()
}
