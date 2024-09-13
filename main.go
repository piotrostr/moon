// main.go
package main

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
)

type MessageType byte

const (
	LatestBlockHashMessageType MessageType = 0x02
	PairsMessageType           MessageType = 0x00
	PingMessageType            MessageType = 0x22 // New message type
)

type LatestBlockHashMessage struct {
	Version     string
	Endpoint    string
	LatestBlock uint32
	Hash        [32]byte
}

type PairsMessage struct {
	Version      string
	PairsCount   uint32
	RawPairsData []byte
}

type PairData struct {
	PairAddress     []byte
	TokenName       string
	TokenSymbol     string
	BaseTokenSymbol string
	Price           float64
	Volume          float64
}

type PingMessage struct {
	Content string
}

func (m *LatestBlockHashMessage) UnmarshalBinary(data []byte) error {
	if len(data) < 36 {
		return errors.New("insufficient data for LatestBlockHashMessage")
	}

	versionEnd := strings.IndexByte(string(data[2:]), 0)
	if versionEnd == -1 {
		return errors.New("invalid version string")
	}
	m.Version = string(data[2 : 2+versionEnd])

	endpointStart := 2 + versionEnd + 1
	endpointEnd := strings.IndexByte(string(data[endpointStart:]), 0)
	if endpointEnd == -1 {
		m.Endpoint = ""
	} else {
		m.Endpoint = string(data[endpointStart : endpointStart+endpointEnd])
	}

	hashStart := len(data) - 36
	m.LatestBlock = binary.LittleEndian.Uint32(data[hashStart : hashStart+4])
	copy(m.Hash[:], data[hashStart+4:])

	return nil
}

func (m *PairsMessage) UnmarshalBinary(data []byte) error {
	if len(data) < 11 {
		return errors.New("insufficient data for PairsMessage")
	}

	versionEnd := strings.IndexByte(string(data[2:]), 0)
	if versionEnd == -1 {
		return errors.New("invalid version string")
	}
	m.Version = string(data[2 : 2+versionEnd])

	pairsStart := 2 + versionEnd + 1
	m.PairsCount = binary.LittleEndian.Uint32(data[pairsStart : pairsStart+4])
	m.RawPairsData = data[pairsStart+4:]

	return nil
}

func (p *PairData) UnmarshalBinary(data []byte) (int, error) {
	if len(data) < 64 {
		return 0, errors.New("insufficient data for PairData")
	}

	p.PairAddress = make([]byte, 32)
	copy(p.PairAddress, data[:32])

	current := 32

	// Helper function to read null-terminated string
	readString := func() (string, int, error) {
		end := strings.IndexByte(string(data[current:]), 0)
		if end == -1 {
			return "", 0, errors.New("invalid string")
		}
		s := string(data[current : current+end])
		return s, current + end + 1, nil
	}

	var err error
	var next int

	p.TokenName, next, err = readString()
	if err != nil {
		return 0, err
	}
	current = next

	p.TokenSymbol, next, err = readString()
	if err != nil {
		return 0, err
	}
	current = next

	p.BaseTokenSymbol, next, err = readString()
	if err != nil {
		return 0, err
	}
	current = next

	if len(data[current:]) < 16 {
		return 0, errors.New("insufficient data for price and volume")
	}

	p.Price = math.Float64frombits(binary.LittleEndian.Uint64(data[current:]))
	p.Volume = math.Float64frombits(binary.LittleEndian.Uint64(data[current+8:]))

	return current + 16, nil
}

func (m *PingMessage) UnmarshalBinary(data []byte) error {
	m.Content = string(data[1:])
	return nil
}

func parseMessage(message []byte) (interface{}, error) {
	if len(message) == 0 {
		return nil, errors.New("empty message")
	}

	fmt.Printf("Message type: %d, Length: %d\n", message[0], len(message))
	fmt.Printf("First 20 bytes: %s\n", hex.EncodeToString(message[:20]))

	switch MessageType(message[0]) {
	case LatestBlockHashMessageType:
		var lbhm LatestBlockHashMessage
		err := lbhm.UnmarshalBinary(message)
		return &lbhm, err
	case PairsMessageType:
		var pm PairsMessage
		err := pm.UnmarshalBinary(message)
		return &pm, err
	case PingMessageType:
		var ping PingMessage
		err := ping.UnmarshalBinary(message)
		return &ping, err
	default:
		return nil, fmt.Errorf("unknown message type: %d", message[0])
	}
}

func main() {
	messageChan := make(chan []byte)
	errorChan := make(chan error)

	go connectWebSocket(messageChan, errorChan)

	for {
		select {
		case message := <-messageChan:
			parsedMessage, err := parseMessage(message)
			if err != nil {
				fmt.Println("Error parsing message:", err)
			} else {
				switch msg := parsedMessage.(type) {
				case *LatestBlockHashMessage:
					fmt.Printf("Received latest block hash: Version=%s, Endpoint=%s, LatestBlock=%d, Hash=%s\n",
						msg.Version, msg.Endpoint, msg.LatestBlock, hex.EncodeToString(msg.Hash[:]))
				case *PairsMessage:
					fmt.Printf("Received pairs message: Version=%s, Number of pairs=%d, Raw data length=%d\n",
						msg.Version, msg.PairsCount, len(msg.RawPairsData))

					// Parse first pair as an example
					if len(msg.RawPairsData) > 0 {
						var pair PairData
						_, err := pair.UnmarshalBinary(msg.RawPairsData)
						if err != nil {
							fmt.Println("Error parsing first pair:", err)
						} else {
							fmt.Printf("First pair: Name=%s, Symbol=%s, BaseSymbol=%s, Price=%f, Volume=%f\n",
								pair.TokenName, pair.TokenSymbol, pair.BaseTokenSymbol, pair.Price, pair.Volume)
						}
					}
				case *PingMessage:
					fmt.Printf("Received ping message: %s\n", msg.Content)
				default:
					fmt.Printf("Received unknown message type: %T\n", msg)
				}
			}
		case err := <-errorChan:
			log.Println("Error:", err)
			return
		}
	}
}
