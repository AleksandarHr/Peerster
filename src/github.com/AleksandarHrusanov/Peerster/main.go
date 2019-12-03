package main

import (
	"flag"
	"strings"

	"github.com/AleksandarHrusanov/Peerster/core"
	"github.com/AleksandarHrusanov/Peerster/gossiper"
	"github.com/AleksandarHrusanov/Peerster/helpers"
	"github.com/AleksandarHrusanov/Peerster/server"
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
	routeRumorPtr := flag.Int("rtimer", 0,
		"Use the given time period in seconds to send a route rumor.")
	// hw3ex2Ptr := flag.Bool("hw3ex2", false,
	// 	"Run gossiper in mode for hw3ex2")
	nPtr := flag.Int("N", 1,
		"Number of nodes in the network, including current peer.")
	stubbornTimeoutPtr := flag.Int("stubbornTimeout", 5,
		"Resend tlc message if no majority of acks before that many seconds.")
	// hopLimitPtr := flag.Int("hopLimit", 10,
	// "Hop limit for TLCAck")
	flag.Parse()

	// Check that the gossiper has a name
	if strings.Compare(*namePtr, "") == 0 {
		panic("Peerster must have a name!")
	}

	// Remove eventual duplicate addresses
	knownPeers := gossiper.CreateSliceKnownPeers(*peersPtr)
	knownPeers = helpers.VerifyRemoveDuplicateAddrInSlice(knownPeers)

	// Create and start gossiper
	gossiperPtr := core.NewGossiper(*gossipAddrPtr,
		*namePtr,
		knownPeers,
		*UIPortPtr)

	// Start server
	go server.StartServer(gossiperPtr)
	gossiper.StartGossiper(gossiperPtr, simplePtr, antiEntropyPtr, routeRumorPtr, nPtr, stubbornTimeoutPtr)
}
