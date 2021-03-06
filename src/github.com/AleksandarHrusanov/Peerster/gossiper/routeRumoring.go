package gossiper

import (
	"strings"
	"time"

	"github.com/AleksandarHrusanov/Peerster/core"
	"github.com/AleksandarHrusanov/Peerster/helpers"
	"github.com/dedis/protobuf"
)

// A function to generate a route rumor (e.g. with empty Text field)
//		and send it to a randomly chosen known peer
func generateAndSendRouteRumor(gossiperPtr *core.Gossiper, toAll bool) {

	gossiperPtr.MongeringIDLock.Lock()
	rumorOrigin := gossiperPtr.Name
	rumorID := gossiperPtr.CurrentMongeringID

	newRouteRumor := core.RumorMessage{
		Origin: rumorOrigin,
		ID:     rumorID,
		Text:   "",
	}
	gossiperPtr.CurrentMongeringID++
	gossiperPtr.MongeringIDLock.Unlock()

	packetToSend := core.GossipPacket{Rumor: &newRouteRumor}
	packetBytes, err := protobuf.Encode(&packetToSend)
	helpers.HandleErrorFatal(err)

	gossiperPtr.PeersLock.Lock()
	knownPeers := gossiperPtr.KnownPeers
	gossiperPtr.PeersLock.Unlock()

	if toAll {
		for _, peer := range knownPeers {
			core.ConnectAndSend(peer, gossiperPtr.Conn, packetBytes)
		}
	} else {
		chosenAddr := ""
		if len(knownPeers) > 0 {
			chosenAddr = helpers.PickRandomInSlice(knownPeers)
		}

		if strings.Compare(chosenAddr, "") != 0 {
			core.ConnectAndSend(chosenAddr, gossiperPtr.Conn, packetBytes)
		}
	}
}

// A function which sends the initial route rumor on start up and then sends
//		new route rumor periodically based on a user-specified flag
func routeRumorHandler(gossiperPtr *core.Gossiper, routeRumorPtr *int) {
	if *routeRumorPtr > 0 {
		// if the route rumor timer is 0, disable sending route rumors completely
		generateAndSendRouteRumor(gossiperPtr, true)
		for {
			time.Sleep(time.Duration(*routeRumorPtr) * time.Second)
			generateAndSendRouteRumor(gossiperPtr, false)
		}
	}
}
