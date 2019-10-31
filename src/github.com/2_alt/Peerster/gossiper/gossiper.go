package gossiper

import (
	"math/rand"
	"time"

	"github.com/2_alt/Peerster/core"
	"github.com/2_alt/Peerster/helpers"
)

// StartGossiper Start the gossiper
func StartGossiper(gossiperPtr *core.Gossiper, simplePtr *bool, antiEntropyPtr *int, routeRumorPtr *int) {
	rand.Seed(time.Now().UnixNano())

	// Listen from client and peers
	if !*simplePtr {
		go clientListener(gossiperPtr, *simplePtr)
		go peersListener(gossiperPtr, *simplePtr)
	} else {
		// In simple mode there is no anti-entropy so no infinite loop
		// to prevent the program to end
		go clientListener(gossiperPtr, *simplePtr)
		peersListener(gossiperPtr, *simplePtr)
	}
	defer gossiperPtr.Conn.Close()
	defer gossiperPtr.LocalConn.Close()

	// Send the initial route rumor message on startup
	go routeRumorHandler(gossiperPtr, routeRumorPtr)

	// Anti-entropy
	if *antiEntropyPtr > 0 {
		for {
			time.Sleep(time.Duration(*antiEntropyPtr) * time.Second)

			if len(gossiperPtr.KnownPeers) > 0 {
				randomAddress := helpers.PickRandomInSlice(gossiperPtr.KnownPeers)
				sendStatus(gossiperPtr, randomAddress)
			}
		}
	} else {
		for {
			time.Sleep(time.Duration(*antiEntropyPtr) * 999)
		}
	}
}
