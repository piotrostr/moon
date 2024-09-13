package main

import (
	"encoding/hex"
	"fmt"

	"github.com/fatih/color"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func logMessageInfo(msgType MessageType, msgSize int, message []byte) {
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
}

func printLatestBlockHashMessage(msg *LatestBlockHashMessage) {
	color.Cyan("Received latest block hash: Version=%s, Endpoint=%s, LatestBlock=%d, Hash=%s",
		msg.Version, msg.Endpoint, msg.LatestBlock, hex.EncodeToString(msg.Hash[:]))
}

func printPairsMessage(msg *PairsMessage) {
	color.Green("Received pairs message: Version=%s, Number of pairs=%d", msg.Version, len(msg.Pairs))

	for i, pair := range msg.Pairs[:min(5, len(msg.Pairs))] {
		color.Green("Pair %d:", i)
		color.Green("  PairAddress: %s", hex.EncodeToString(pair.PairAddress[:]))
		color.Green("  TokenName: %s", pair.TokenName)
		color.Green("  TokenSymbol: %s", pair.TokenSymbol)
		color.Green("  BaseTokenSymbol: %s", pair.BaseTokenSymbol)
		color.Green("  Price: %f", pair.Price)
		color.Green("  Volume: %f", pair.Volume)
	}
}

func printPingMessage(msg *PingMessage) {
	color.Yellow("Received ping message: %s", msg.Content)
}
