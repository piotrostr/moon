// main.go
package main

import (
	"github.com/fatih/color"
)

func main() {
	messageChan := make(chan []byte)
	errorChan := make(chan error)

	go connectWebSocket(messageChan, errorChan)

	for {
		select {
		case message := <-messageChan:
			if err := handleMessage(message); err != nil {
				color.Red("Error handling message: %v", err)
			}
		case err := <-errorChan:
			color.Red("WebSocket error: %v", err)
			return
		}
	}
}

func handleMessage(message []byte) error {
	parsedMessage, err := parseMessage(message)
	if err != nil {
		return err
	}

	switch msg := parsedMessage.(type) {
	case *LatestBlockHashMessage:
		printLatestBlockHashMessage(msg)
	case *PairsMessage:
		printPairsMessage(msg)
	case *PingMessage:
		printPingMessage(msg)
	default:
		color.Red("Received unknown message type: %T", msg)
	}

	return nil
}
