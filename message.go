package main

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/fatih/color"
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

type PingMessage struct {
	Content string
}

func (m *PingMessage) UnmarshalBinary(data []byte) error {
	m.Content = string(data)
	return nil
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

type PairsMessage struct {
	Version string
	Pairs   []PairData
}

type PairData struct {
	PairAddress     [32]byte
	UnknownData     [32]byte
	TokenName       string
	TokenSymbol     string
	BaseTokenSymbol string
	Price           float64
	Volume          float64
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
	pairsData := data[pairsStart:]

	for len(pairsData) >= 64 {
		var pair PairData
		bytesRead, err := pair.UnmarshalBinary(pairsData)
		if err != nil {
			return err
		}
		m.Pairs = append(m.Pairs, pair)
		pairsData = pairsData[bytesRead:]
	}

	return nil
}

func (p *PairData) UnmarshalBinary(data []byte) (int, error) {
	if len(data) < 64 {
		return 0, errors.New("insufficient data for PairData")
	}

	copy(p.PairAddress[:], data[:32])
	copy(p.UnknownData[:], data[32:64])

	current := 64

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

func parseMessage(message []byte) (interface{}, error) {
	if len(message) == 0 {
		return nil, errors.New("empty message")
	}

	msgType := MessageType(message[0])
	msgSize := len(message)

	switch msgType {
	case LatestBlockHashMessageType:
		color.Cyan("Message type: LatestBlockHash (0x%02x), Size: %d bytes", msgType, msgSize)
	case PairsMessageType:
		color.Green("Message type: Pairs (0x%02x), Size: %d bytes", msgType, msgSize)
	case PingMessageType:
		color.Yellow("Message type: Ping (0x%02x), Size: %d bytes", msgType, msgSize)
	default:
		color.Red("Unknown message type: 0x%02x, Size: %d bytes", msgType, msgSize)
	}

	fmt.Printf("First 20 bytes: %s\n", hex.EncodeToString(message[:min(20, len(message))]))

	switch msgType {
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
