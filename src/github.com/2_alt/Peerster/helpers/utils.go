package helpers

import (
	"fmt"
	"math/rand"
	"net"
	"strings"
	"github.com/dedis/protobuf"
	"github.com/2_alt/Peerster/core"
)

// HandleErrorFatal Handle an error
func HandleErrorFatal(err error) {
	if err != nil {
		panic("Error: " + err.Error())
	}
}

// HandleErrorNonFatal Handle an error without stopping the program
func HandleErrorNonFatal(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

// ClientConnectAndSend connects to the given gossiper's address and send the text to it.
// This function is used by the server to send a message to the gossiper.
func ClientConnectAndSend(remoteAddr string, text *string) {
	// Create GossipPacket and encapsulate message into it
	msg := &core.Message{Text: *text}
	packetBytes, err := protobuf.Encode(msg)
	HandleErrorFatal(err)

	// Connect
	dst, err := net.ResolveUDPAddr("udp4", remoteAddr)
	_ = dst
	HandleErrorFatal(err)
	conn, err := net.DialUDP("udp4", nil, dst)
	HandleErrorFatal(err)
	defer conn.Close()

	// Send packet
	_, err = conn.Write(packetBytes)
	HandleErrorFatal(err)
}

// ConnectAndSend Connect to the given address and send the packet
func ConnectAndSend(addressAndPort string, conn *net.UDPConn, packetToSend []byte) {
	// If a Peerster does not know any other Peers, the address can be an empty string
	if strings.Compare(addressAndPort, "") == 0 {
		return
	}

	// Resolve destination address
	dst, err := net.ResolveUDPAddr("udp4", addressAndPort)
	HandleErrorFatal(err)

	// Send packet
	_, err = conn.WriteToUDP(packetToSend, dst)
	HandleErrorFatal(err)
}

// CreateStringKnownPeers Create a string from the map of known peers
func CreateStringKnownPeers(knownPeers []string) string {
	stringPeers := ""
	if len(knownPeers) == 0 {
		return stringPeers
	}
	for _, peerAddress := range knownPeers {
		stringPeers += peerAddress
		stringPeers += ","
	}
	stringPeers = stringPeers[:len(stringPeers)-1]
	return stringPeers
}

// SliceContainsString Check if a slice contains a string
func SliceContainsString(sx []string, s string) bool {
	for _, str := range sx {
		if strings.Compare(str, s) == 0 {
			return true
		}
	}
	return false
}

// MapContainsString Check if a map contains a string in its keys
func MapContainsString(m map[string]uint32, s string) bool {
	for st := range m {
		if strings.Compare(st, s) == 0 {
			return true
		}
	}
	return false
}

// PickRandomInMap Pick a random key in a map
func PickRandomInMap(m map[string]uint32) string {
	var list []string
	for mapElem := range m {
		list = append(list, mapElem)
	}
	return PickRandomInSlice(list)
}

// PickRandomInSlice Pick a random string in a []string
func PickRandomInSlice(sx []string) string {
	if (len(sx)) == 0 {
		return ""
	}
	randomInt := rand.Intn(len(sx))
	return sx[randomInt]
}

// PickRandomInSliceDifferentFrom Pick a random string in a []string different from a given string
func PickRandomInSliceDifferentFrom(sx []string, notThisOne string) string {
	if len(sx) <= 1 {
		return ""
	}
	currentAddr := notThisOne
	for strings.Compare(notThisOne, currentAddr) == 0 {
		currentAddr = PickRandomInSlice(sx)
	}
	return currentAddr
}

// ContainsRumor Check if a list contains a Rumor, return it if true
func ContainsRumor(list []core.RumorMessage, r *core.RumorMessage) bool {
	for _, rm := range list {
		if strings.Compare(r.Origin, rm.Origin) == 0 && r.ID == rm.ID {
			return true
		}
	}
	return false
}

// IsRumorOriginKnown Check if a Rumor Origin is known and return the last Rumor ID we know
func IsRumorOriginKnown(list []core.RumorMessage, r *core.RumorMessage) (bool, uint32) {
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
func IsRumorKnown(listOfWanted []core.PeerStatus, r *core.RumorMessage) (bool, bool, uint32) {
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

// IPAddressIsValid Check if a given ip4 address is valid
func IPAddressIsValid(address string) bool {
	_, err := net.ResolveUDPAddr("udp4", address)
	return err == nil
}

// VerifyRemoveDuplicateAddrInSlice Remove duplicate addresses in a slice of strings.
// It also removes all invalid ip4 addresses from the list
func VerifyRemoveDuplicateAddrInSlice(sx []string) []string {
	newList := make([]string, 0)
	mapElem := make(map[string]bool)
	for _, address := range sx {
		_, present := mapElem[address]
		if !present && IPAddressIsValid(address) {
			mapElem[address] = true
			newList = append(newList, address)
		}
	}
	return newList
}

// ========================================================
// ========================================================
//						Homework 2 functions
// ========================================================
// ========================================================

//UpdateDestinationTable - a function to update destination table (if needed) on receiving a rumor
func UpdateDestinationTable(rumorOrigin string, rumorID uint32, fromAddr string,
	destinationTable map[string]string, knownRumors []core.RumorMessage, originIsKnown bool) {

	if !originIsKnown {
		// First rumor from Origin
		destinationTable[rumorOrigin] = fromAddr
		PrintOutputUpdatingDSDV(rumorOrigin, fromAddr)
	} else {
		// Update table if the sequence number of the rumor is greater than any known rumors' ID
		//	from the same origin
		toUpdate := true
		for _, r := range knownRumors {
			if strings.Compare(r.Origin, rumorOrigin) == 0 {
				if r.ID > rumorID {
					toUpdate = false
				}
			}
		}

		if toUpdate {
			destinationTable[rumorOrigin] = fromAddr
			PrintOutputUpdatingDSDV(rumorOrigin, fromAddr)
		}
	}
}
