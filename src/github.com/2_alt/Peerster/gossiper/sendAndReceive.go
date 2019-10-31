package gossiper

import (
	"net"
	"strings"
	"time"

	"github.com/2_alt/Peerster/core"
	"github.com/2_alt/Peerster/helpers"
	"github.com/dedis/protobuf"
)

// Send the status of the Gossiper to the given address
func sendStatus(gossiper *core.Gossiper, toAddr string) {
	if strings.Compare(toAddr, "") == 0 {
		panic("ERROR")
	}
	sp := core.StatusPacket{Want: gossiper.Want}
	packetToSend := core.GossipPacket{Status: &sp}
	packetBytes, err := protobuf.Encode(&packetToSend)
	helpers.HandleErrorFatal(err)
	core.ConnectAndSend(toAddr, gossiper.Conn, packetBytes)
}

// Send a RumorMessage to the given address
func sendRumor(r core.RumorMessage, gossiper *core.Gossiper, toAddr string) {
	packetToSend := core.GossipPacket{Rumor: &r}
	packetBytes, err := protobuf.Encode(&packetToSend)

	// Create new mongering status, set a timer for it and
	// append it to the slice in the gossiper struct
	newMongeringStatus := core.MongeringStatus{
		RumorMessage:          r,
		WaitingStatusFromAddr: toAddr,
		TimeUp:                make(chan bool),
		AckReceived:           false,
	}
	go func(mongeringStatusPtr *core.MongeringStatus) {
		time.Sleep(10 * time.Second)
		mongeringStatusPtr.TimeUp <- true
	}(&newMongeringStatus)
	gossiper.MongeringStatus = append(gossiper.MongeringStatus, &newMongeringStatus)

	helpers.HandleErrorFatal(err)
	core.ConnectAndSend(toAddr, gossiper.Conn, packetBytes)
}

// Receive a message from UDP and decode it into a GossipPacket
func receiveAndDecode(gossiper *core.Gossiper) (core.GossipPacket, *net.UDPAddr) {
	// Create buffer
	buffer := make([]byte, 9216)

	// Read message from UDP
	conn := gossiper.Conn
	size, fromAddr, err := conn.ReadFromUDP(buffer)

	// Timeout
	helpers.HandleErrorNonFatal(err)
	if err != nil {
		return core.GossipPacket{}, nil
	}

	// Decode the packet
	gossipPacket := core.GossipPacket{}
	err = protobuf.Decode(buffer[0:size], &gossipPacket)
	helpers.HandleErrorNonFatal(err)

	return gossipPacket, fromAddr
}

// Receive a client's message from UDP and decode it into a GossipPacket
func receiveAndDecodeFromClient(gossiper *core.Gossiper) (core.Message, *net.UDPAddr) {
	// Create buffer
	buffer := make([]byte, 128)

	// Read message from UDP
	conn := gossiper.LocalConn
	size, fromAddr, err := conn.ReadFromUDP(buffer)

	// Timeout
	helpers.HandleErrorNonFatal(err)
	if err != nil {
		return core.Message{}, nil
	}

	// Decode the packet
	message := core.Message{}
	err = protobuf.Decode(buffer[0:size], &message)
	helpers.HandleErrorNonFatal(err)

	return message, fromAddr
}
