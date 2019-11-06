package gossiper

import (
	"strings"

	"github.com/AleksandarHrusanov/Peerster/core"
	"github.com/AleksandarHrusanov/Peerster/helpers"
	"github.com/dedis/protobuf"
)

// Given a message from the client, return true if it is private
func isClientMessagePrivate(clientMsg *core.Message) bool {
	return (strings.Compare(*(clientMsg.Destination), "") != 0)
}

// A constructor for PrivateMessages - defaultID = 0 and defaultHopLimit = 10
func createNewPrivateMessage(origin string, msg string, dest *string) *core.PrivateMessage {
	defaultID := uint32(0) // to enforce NOT sequencing
	defaultHopLimit := uint32(10)
	privateMsg := core.PrivateMessage{Origin: origin, ID: defaultID, Text: msg, Destination: *dest, HopLimit: defaultHopLimit}
	return &privateMsg
}

func handlePrivateMessage(gossiper *core.Gossiper, privateMsg *core.PrivateMessage) {
	if privateMessageReachedDestination(gossiper, privateMsg) {
		// If private message reached its destination, print to console
		helpers.PrintOutputPrivateMessage(privateMsg.Origin, privateMsg.HopLimit, privateMsg.Text)
	} else {
		// If this is not the private message's destination, forward message to next hop
		forwardPrivateMessage(gossiper, privateMsg)
	}
}

// Given a private message, returns true if the current gossiper is it's destination
func privateMessageReachedDestination(gossiperPtr *core.Gossiper, msg *core.PrivateMessage) bool {
	return (strings.Compare(gossiperPtr.Name, msg.Destination) == 0)
}

// A function to forward a private message to the corresponding next hop
func forwardPrivateMessage(gossiperPtr *core.Gossiper, msg *core.PrivateMessage) {

	if msg.HopLimit == 0 {
		// if we have reached the HopLimit, drop the message
		return
	}

	gossiperPtr.DestinationTable.DsdvLock.Lock()
	forwardingAddress := gossiperPtr.DestinationTable.Dsdv[msg.Destination]
	gossiperPtr.DestinationTable.DsdvLock.Unlock()
	// If current node has no information about next hop to the destination in question
	if strings.Compare(forwardingAddress, "") == 0 {
		// TODO: What to do if there is no 'next hop' known when peer has to forward a private packet
	}

	// Decrement the HopLimit right before forwarding the packet
	msg.HopLimit--
	// Encode and send packet
	packetToSend := core.GossipPacket{Private: msg}
	packetBytes, err := protobuf.Encode(&packetToSend)
	helpers.HandleErrorFatal(err)
	core.ConnectAndSend(forwardingAddress, gossiperPtr.Conn, packetBytes)
}
