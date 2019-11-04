package helpers

import (
	"fmt"
	"strconv"
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
func PrintOutputRumorFromPeer(origin string, rumorFrom string, id uint32, text string, knownPeers []string) {
	// Print first line
	fmt.Println("RUMOR origin " + origin +
		" from " + rumorFrom +
		" ID " + strconv.Itoa(int(id)) +
		" contents " + text)

	// Print second line
	stringPeers := CreateStringKnownPeers(knownPeers)
	fmt.Println("PEERS " + stringPeers)
}

// PrintOutputMongering print on the console
func PrintOutputMongering(withAddr string) {
	fmt.Println("MONGERING with " + withAddr)
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
	fmt.Printf("DSDV %s %s\n", peerName, ipPort)
}

//PrintOutputPrivateMessage print to console
func PrintOutputPrivateMessage(origin string, hopLimit uint32, contents string) {
	fmt.Printf("PRIVATE origin %s hop-limit %d contents %s", origin, hopLimit, contents)
}

// PrintDownloadingMetafile print to console
func PrintDownloadingMetafile(fname string, downloadFrom string) {
	fmt.Printf("DOWNLOADING metafile of %s from %s\n", fname, downloadFrom)
}

// PrintDownloadingChunk print to console
func PrintDownloadingChunk(fname string, downloadFrom string, idx uint32) {
	startFromOne := idx + 1
	fmt.Printf("DOWNLOADING %s chunk %d from %s\n", fname, startFromOne, downloadFrom)
}

// PrintReconstructedFile print to console
func PrintReconstructedFile(fname string) {
	fmt.Printf("RECONSTRUCTED file %s\n", fname)
}
