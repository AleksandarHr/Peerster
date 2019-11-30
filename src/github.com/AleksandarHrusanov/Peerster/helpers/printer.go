package helpers

import (
	"fmt"
)

// PrintOutputSimpleMessageFromClient print on the console
func PrintOutputSimpleMessageFromClient(messageText string, knownPeers []string) {
	// Print first line
	fmt.Printf("CLIENT MESSAGE %s\n", messageText)

	// Print second line
	stringPeers := CreateStringKnownPeers(knownPeers)
	fmt.Printf("PEERS %s\n", stringPeers)
}

// PrintOutputSimpleMessageFromPeer print on the console
func PrintOutputSimpleMessageFromPeer(messageText string, senderName string,
	relayAddr string, knownPeers []string) {
	// Print first line
	fmt.Printf("SIMPLE MESSAGE origin %s from %s contents %s\n", senderName, relayAddr, messageText)

	// Print second line
	stringPeers := CreateStringKnownPeers(knownPeers)
	fmt.Printf("PEERS %s\n", stringPeers)
}

// PrintOutputRumorFromPeer print on the console
func PrintOutputRumorFromPeer(origin string, rumorFrom string, id uint32, text string, knownPeers []string) {
	// Print first line
	fmt.Printf("RUMOR origin %s from %s ID %d contents %s\n", origin, rumorFrom, id, text)

	// Print second line
	// stringPeers := CreateStringKnownPeers(knownPeers)
	// fmt.Println("PEERS " + stringPeers)
}

// PrintOutputMongering print on the console
func PrintOutputMongering(withAddr string) {
	fmt.Printf("MONGERING with %s\n", withAddr)
}

// PrintOutputInSyncWith print on the console
func PrintOutputInSyncWith(addr string) {
	fmt.Printf("IN SYNC WITH %s\n", addr)
}

// PrintOutputFlippedCoin print on the console
func PrintOutputFlippedCoin(addr string) {
	fmt.Printf("FLIPPED COIN sending rumor to %s\n", addr)
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
	fmt.Printf("PRIVATE origin %s hop-limit %d contents %s\n", origin, hopLimit, contents)
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

// ========================================================
// ========================================================
//						Homework 3 functions
// ========================================================
// ========================================================

func PrintFileMatchFound(fname string, peer string, metahash string, chunks string) {
	fmt.Printf("FOUND match %s at %s metafile=%s chunks=%s\n", fname, peer, metahash, chunks)
}

func PrintSearchFinished() {
	fmt.Println("SEARCH FINISHED")
}
