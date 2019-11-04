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
func generateAndSendRouteRumor(gossiperPtr *core.Gossiper, rumorOrigin string, rumorID uint32) {
	chosenAddr := ""
	if len(gossiperPtr.KnownPeers) > 0 {
		chosenAddr = helpers.PickRandomInSlice(gossiperPtr.KnownPeers)
	}

	if strings.Compare(chosenAddr, "") != 0 {
		newRouteRumor := core.RumorMessage{
			Origin: rumorOrigin,
			ID:     rumorID,
			Text:   "",
		}
		packetToSend := core.GossipPacket{Rumor: &newRouteRumor}
		packetBytes, err := protobuf.Encode(&packetToSend)
		helpers.HandleErrorFatal(err)
		core.ConnectAndSend(chosenAddr, gossiperPtr.Conn, packetBytes)
	}
}

// A function which sends the initial route rumor on start up and then sends
//		new route rumor periodically based on a user-specified flag
func routeRumorHandler(gossiperPtr *core.Gossiper, routeRumorPtr *int) {
	if *routeRumorPtr > 0 {
		// if the route rumor timer is 0, disable sending route rumors completely
		generateAndSendRouteRumor(gossiperPtr, gossiperPtr.Name, gossiperPtr.CurrentRumorID)
		for {
			time.Sleep(time.Duration(*routeRumorPtr) * time.Second)
			generateAndSendRouteRumor(gossiperPtr, gossiperPtr.Name, gossiperPtr.CurrentRumorID)
			gossiperPtr.CurrentRumorID++
		}
	}
}
