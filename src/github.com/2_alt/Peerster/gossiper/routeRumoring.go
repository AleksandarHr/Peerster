package gossiper

import (
	"strings"
	"time"

	"github.com/2_alt/Peerster/core"
	"github.com/2_alt/Peerster/helpers"
	"github.com/dedis/protobuf"
)

// A function to generate a route rumor (e.g. with empty Text field)
//		and send it to a randomly chosen known peer
func generateAndSendRouteRumor(gossiperPtr *core.Gossiper, toAll bool) {

	gossiperPtr.RumorIDLock.Lock()
	rumorOrigin := gossiperPtr.Name
	rumorID := gossiperPtr.CurrentRumorID

	newRouteRumor := core.RumorMessage{
		Origin: rumorOrigin,
		ID:     rumorID,
		Text:   "",
	}
	gossiperPtr.CurrentRumorID++
	gossiperPtr.RumorIDLock.Unlock()

	packetToSend := core.GossipPacket{Rumor: &newRouteRumor}
	packetBytes, err := protobuf.Encode(&packetToSend)
	helpers.HandleErrorFatal(err)

	if toAll {
		for _, peer := range gossiperPtr.KnownPeers {
			core.ConnectAndSend(peer, gossiperPtr.Conn, packetBytes)
		}
	} else {
		chosenAddr := ""
		if len(gossiperPtr.KnownPeers) > 0 {
			chosenAddr = helpers.PickRandomInSlice(gossiperPtr.KnownPeers)
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
