package helpers

import (
	"fmt"
	"strconv"
	"github.com/2_alt/Peerster/core"
)

// PrintOutputSimpleMessageFromClient print on the console
func PrintOutputSimpleMessageFromClient(messageText string, knownPeers []string) {
	// Print first line
	fmt.Println("CLIENT MESSAGE " + messageText)

	// Print second line
	stringPeers := CreateStringKnownPeers(knownPeers)
	fmt.Println("PEERS " + stringPeers)
}

// PrintOutputSimpleMessageFromPeer print on the console
func PrintOutputSimpleMessageFromPeer(messageText string, senderName string,
	relayAddr string, knownPeers []string) {
	// Print first line
	fmt.Println("SIMPLE MESSAGE origin " + senderName +
		" from " + relayAddr +
		" contents " + messageText)

	// Print second line
	stringPeers := CreateStringKnownPeers(knownPeers)
	fmt.Println("PEERS " + stringPeers)
}

// PrintOutputRumorFromPeer print on the console
func PrintOutputRumorFromPeer(rumor *core.RumorMessage, rumorFrom string, knownPeers []string) {
	// Print first line
	fmt.Println("RUMOR origin " + rumor.Origin +
		" from " + rumorFrom +
		" ID " + strconv.Itoa(int(rumor.ID)) +
		" contents " + rumor.Text)

	// Print second line
	stringPeers := CreateStringKnownPeers(knownPeers)
	fmt.Println("PEERS " + stringPeers)
}

// PrintOutputMongering print on the console
func PrintOutputMongering(withAddr string) {
	fmt.Println("MONGERING with " + withAddr)
}

// PrintOutputStatus print on the console
func PrintOutputStatus(fromAddr string, listOfWanted []core.PeerStatus, knownPeers []string) {
	// Print first line
	fmt.Print("STATUS from " + fromAddr)
	for _, peerStatus := range listOfWanted {
		fmt.Print(" peer " + peerStatus.Identifier + " nextID " + strconv.Itoa(int(peerStatus.NextID)))
	}
	fmt.Println()

	// Print second line
	stringPeers := CreateStringKnownPeers(knownPeers)
	fmt.Println("PEERS " + stringPeers)
}

// PrintOutputInSyncWith print on the console
func PrintOutputInSyncWith(addr string) {
	fmt.Println("IN SYNC WITH " + addr)
}

// PrintOutputFlippedCoin print on the console
func PrintOutputFlippedCoin(addr string) {
	fmt.Println("FLIPPED COIN sending rumor to " + addr)
}


// ========================================================
// ========================================================
//						Homework 2 functions
// ========================================================
// ========================================================

//PrintOutputUpdatingDSDV print on the console
func PrintOutputUpdatingDSDV(peerName string, ipPort string) {
	fmt.Println("DSDV " + peerName + " " + ipPort)
}
