package main

import (
	"flag"
	"strings"
	"github.com/2_alt/Peerster/gossiper"
	"github.com/2_alt/Peerster/helpers"
	"github.com/2_alt/Peerster/server"
)

func main() {
	UIPortPtr := flag.String("UIPort", "8080",
		"Port for the UI client")
	gossipAddrPtr := flag.String("gossipAddr", "127.0.0.1:5000",
		"ip:port for the gossiper")
	namePtr := flag.String("name", "", "name of the gossiper")
	peersPtr := flag.String("peers", "",
		"comma separated list of peers of the form ip:port")
	simplePtr := flag.Bool("simple", false,
		"run gossiper in simple broadcast mode")
	antiEntropyPtr := flag.Int("antiEntropy", 10,
		"Use the given timeout in seconds for anti-entropy.")
	flag.Parse()

	// Check that the gossiper has a name
	if strings.Compare(*namePtr, "") == 0 {
		panic("Peerster must have a name!")
	}

	// Remove eventual duplicate addresses
	knownPeers := gossiper.CreateSliceKnownPeers(*peersPtr)
	knownPeers = helpers.VerifyRemoveDuplicateAddrInSlice(knownPeers)

	// Create and start gossiper
	gossiperPtr := gossiper.NewGossiper(*gossipAddrPtr,
		*namePtr,
		knownPeers,
		*UIPortPtr)

	// Start server
	go server.StartServer(gossiperPtr)
	gossiper.StartGossiper(gossiperPtr, simplePtr, antiEntropyPtr)
}
