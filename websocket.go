// websocket.go
package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

func connectWebSocket(messageChan chan<- []byte, errorChan chan<- error) {
	url := "wss://io.dexscreener.com/dex/screener/v4/pairs/h24/1?rankBy[key]=pairAge&rankBy[order]=asc&filters[chainIds][0]=solana&filters[dexIds][0]=moonshot&filters[excludedDexIds][]&filters[moonshotProgress][max]=99.99"
	fmt.Println("Connecting to:", url)

	dialer := websocket.Dialer{
		EnableCompression: false,
	}

	header := http.Header{}
	header.Set("Origin", "https://dexscreener.com")
	header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36")

	conn, _, err := dialer.Dial(url, header)
	if err != nil {
		errorChan <- fmt.Errorf("WebSocket connection error: %v", err)
		return
	}
	defer conn.Close()

	fmt.Println("WebSocket connection opened")

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			errorChan <- fmt.Errorf("WebSocket read error: %v", err)
			return
		}
		messageChan <- message
	}
}
