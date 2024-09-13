// main.go
package main

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"unsafe"
)

type MessageType byte

const (
	LatestBlockHashMessageType MessageType = 0x02
	PairsMessageType           MessageType = 0x00
)

type LatestBlockHashMessage struct {
	Version     string
	Endpoint    string
	LatestBlock uint32
	Hash        [32]byte
}

type PairsMessage struct {
	Version string
	Pairs   []PairData
}

type PairData struct {
	PairAddress     [32]byte
	TokenName       string
	TokenSymbol     string
	BaseTokenSymbol string
	Price           float64
	Volume          float64
	// Add other fields as needed
}

func (m *LatestBlockHashMessage) UnmarshalBinary(data []byte) error {
	if len(data) < 73 {
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
		return errors.New("invalid endpoint string")
	}
	m.Endpoint = string(data[endpointStart : endpointStart+endpointEnd])

	blockStart := endpointStart + endpointEnd + 1
	m.LatestBlock = binary.LittleEndian.Uint32(data[blockStart : blockStart+4])

	hashStart := blockStart + 4
	copy(m.Hash[:], data[hashStart:hashStart+32])

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
	for pairsStart < len(data) {
		var pair PairData
		pairEnd, err := pair.UnmarshalBinary(data[pairsStart:])
		if err != nil {
			return err
		}
		m.Pairs = append(m.Pairs, pair)
		pairsStart += pairEnd
	}

	return nil
}

func (p *PairData) UnmarshalBinary(data []byte) (int, error) {
	if len(data) < 64 {
		return 0, errors.New("insufficient data for PairData")
	}

	copy(p.PairAddress[:], data[:32])

	nameEnd := strings.IndexByte(string(data[32:]), 0)
	if nameEnd == -1 {
		return 0, errors.New("invalid token name")
	}
	p.TokenName = string(data[32 : 32+nameEnd])

	symbolStart := 32 + nameEnd + 1
	symbolEnd := strings.IndexByte(string(data[symbolStart:]), 0)
	if symbolEnd == -1 {
		return 0, errors.New("invalid token symbol")
	}
	p.TokenSymbol = string(data[symbolStart : symbolStart+symbolEnd])

	baseSymbolStart := symbolStart + symbolEnd + 1
	baseSymbolEnd := strings.IndexByte(string(data[baseSymbolStart:]), 0)
	if baseSymbolEnd == -1 {
		return 0, errors.New("invalid base token symbol")
	}
	p.BaseTokenSymbol = string(data[baseSymbolStart : baseSymbolStart+baseSymbolEnd])

	priceStart := baseSymbolStart + baseSymbolEnd + 1
	p.Price = *(*float64)(unsafe.Pointer(&data[priceStart]))
	p.Volume = *(*float64)(unsafe.Pointer(&data[priceStart+8]))

	return priceStart + 16, nil
}

func parseMessage(message []byte) (interface{}, error) {
	if len(message) == 0 {
		return nil, errors.New("empty message")
	}

	switch MessageType(message[0]) {
	case LatestBlockHashMessageType:
		var lbhm LatestBlockHashMessage
		err := lbhm.UnmarshalBinary(message)
		return &lbhm, err
	case PairsMessageType:
		var pm PairsMessage
		err := pm.UnmarshalBinary(message)
		return &pm, err
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
					fmt.Printf("Received pairs message: Version=%s, Number of pairs=%d\n", msg.Version, len(msg.Pairs))
					for i, pair := range msg.Pairs {
						fmt.Printf("  Pair %d: Name=%s, Symbol=%s, BaseSymbol=%s, Price=%f, Volume=%f\n",
							i, pair.TokenName, pair.TokenSymbol, pair.BaseTokenSymbol, pair.Price, pair.Volume)
					}
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
