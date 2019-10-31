package core
import "github.com/dedis/protobuf"
import "github.com/2_alt/Peerster/helpers"
import "net"
import "fmt"
import "strconv"
import "strings"
import "encoding/hex"

// ClientConnectAndSend connects to the given gossiper's address and send the text to it.
// This function is used by the server to send a message to the gossiper.
func ClientConnectAndSend(remoteAddr string, text *string, destination *string, fileToShare *string, request *string) {
	// Create GossipPacket and encapsulate message into it
	requestBytes := make([]byte, 0)
	msg := &Message{Text: *text, Destination: destination, File: fileToShare}
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

// IsRumorOriginKnown Check if a Rumor Origin is known and return the last Rumor ID we know
func IsRumorOriginKnown(list []RumorMessage, r *RumorMessage) (bool, uint32) {
	maxLastID := uint32(0)
	originKnown := false
	for _, rm := range list {
		if strings.Compare(r.Origin, rm.Origin) == 0 {
			originKnown = true
			if rm.ID > maxLastID {
				maxLastID = rm.ID
			}
		}
	}
	return originKnown, maxLastID
}

// IsRumorKnown Check if a Rumor or its Origin is known and return the lastID we have from it.
// Returns "rumor is known" bool, "origin is known" bool and the nextID we have for this
// origin.
func IsRumorKnown(listOfWanted []PeerStatus, r *RumorMessage) (bool, bool, uint32) {
	originIsKnown := false
	rumorIsKnown := false
	nextID := uint32(0)
	for _, peerStatus := range listOfWanted {
		if strings.Compare(peerStatus.Identifier, r.Origin) == 0 {
			originIsKnown = true
			if peerStatus.NextID > r.ID {
				rumorIsKnown = true
				nextID = peerStatus.NextID
				return rumorIsKnown, originIsKnown, nextID
			}
			if peerStatus.NextID > nextID {
				nextID = peerStatus.NextID
			}
		}
	}
	return rumorIsKnown, originIsKnown, nextID
}

// ========================================================
// ========================================================
//						Homework 2 functions
// ========================================================
// ========================================================

//UpdateDestinationTable - a function to update destination table (if needed) on receiving a rumor
func UpdateDestinationTable(rumorOrigin string, rumorID uint32, fromAddr string,
	destinationTable map[string]string, knownRumors []RumorMessage, originIsKnown bool, toPrint bool) {

	if !originIsKnown {
		// First rumor from Origin
		destinationTable[rumorOrigin] = fromAddr
		if toPrint {
			helpers.PrintOutputUpdatingDSDV(rumorOrigin, fromAddr)
		}
	} else {
		// Update table if the sequence number of the rumor is greater than any known rumors' ID
		//	from the same origin
		toUpdate := true
		for _, r := range knownRumors {
			if strings.Compare(r.Origin, rumorOrigin) == 0 {
				if r.ID >= rumorID {
					toUpdate = false
				}
			}
		}

		if toUpdate {
			destinationTable[rumorOrigin] = fromAddr
			if toPrint{
				helpers.PrintOutputUpdatingDSDV(rumorOrigin, fromAddr)
			}
		}
	}
}

//IsRouteRumor - a function which returns true if the rumor is a route rumor (e.g. empty Text field)
func IsRouteRumor(rumor *RumorMessage) bool {
	return (strings.Compare(rumor.Text, "") == 0)
}


// PrintOutputStatus print on the console
func PrintOutputStatus(fromAddr string, listOfWanted []PeerStatus, knownPeers []string) {
	// Print first line
	fmt.Print("STATUS from " + fromAddr)
	for _, peerStatus := range listOfWanted {
		fmt.Print(" peer " + peerStatus.Identifier + " nextID " + strconv.Itoa(int(peerStatus.NextID)))
	}
	fmt.Println()

	// Print second line
	stringPeers := helpers.CreateStringKnownPeers(knownPeers)
	fmt.Println("PEERS " + stringPeers)
}
