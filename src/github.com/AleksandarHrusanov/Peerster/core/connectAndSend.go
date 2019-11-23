package core

import (
	"encoding/hex"
	"net"
	"strings"

	"github.com/AleksandarHrusanov/Peerster/helpers"
	"github.com/dedis/protobuf"
)

// ClientConnectAndSend connects to the given gossiper's address and send the text to it.
// This function is used by the server to send a message to the gossiper.
func ClientConnectAndSend(remoteAddr string, text *string, destination *string, fileToShare *string,
	request *string, keywords *string, budget *uint64) {
	// Create GossipPacket and encapsulate message into it
	requestBytes := make([]byte, 0)
	msg := &Message{Text: *text, Destination: destination, File: fileToShare, Keywords: keywords, Budget: budget}
	if strings.Compare(*request, "") != 0 {
		decoded, err := hex.DecodeString(*request)
		if err == nil {
			requestBytes = decoded
		}
	}
	msg.Request = &requestBytes
	packetBytes, err := protobuf.Encode(msg)
	helpers.HandleErrorFatal(err)

	// Connect
	dst, err := net.ResolveUDPAddr("udp4", remoteAddr)
	_ = dst
	helpers.HandleErrorFatal(err)
	conn, err := net.DialUDP("udp4", nil, dst)
	helpers.HandleErrorFatal(err)
	defer conn.Close()

	// Send packet
	_, err = conn.Write(packetBytes)
	helpers.HandleErrorFatal(err)
}

// ConnectAndSend Connect to the given address and send the packet
func ConnectAndSend(addressAndPort string, conn *net.UDPConn, packetToSend []byte) {
	// If a Peerster does not know any other Peers, the address can be an empty string
	if strings.Compare(addressAndPort, "") == 0 {
		return
	}

	// Resolve destination address
	dst, err := net.ResolveUDPAddr("udp4", addressAndPort)
	helpers.HandleErrorFatal(err)

	// Send packet
	_, err = conn.WriteToUDP(packetToSend, dst)
	helpers.HandleErrorFatal(err)
}

// ContainsRumor Check if a list contains a Rumor, return it if true
func ContainsRumor(list []RumorMessage, r *RumorMessage) bool {
	for _, rm := range list {
		if strings.Compare(r.Origin, rm.Origin) == 0 && r.ID == rm.ID {
			return true
		}
	}
	return false
}
