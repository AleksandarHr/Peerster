package gossiper

import (
	"strings"

	"github.com/AleksandarHrusanov/Peerster/core"
)

// Retrieve a Rumor from a list given its Origin and ID
func getRumor(g *core.Gossiper, o string, i uint32) *core.RumorMessage {
	for _, rm := range g.KnownRumors {
		if strings.Compare(o, rm.Origin) == 0 && i == rm.ID {
			return &rm
		}
	}
	return nil
}

// Update a slice of Rumor without duplicates
func updateRumorListNoDuplicates(r core.RumorMessage, list []core.RumorMessage) []core.RumorMessage {
	if len(list) > 0 {
		for _, rumorInList := range list {
			if strings.Compare(r.Origin, rumorInList.Origin) == 0 && rumorInList.ID == r.ID {
				return list
			}
		}
	}
	list = append(list, r)
	return list
}

// Add a rumor to the gossiper's known rumors if it is not already there
func addRumorToKnownRumors(g *core.Gossiper, r core.RumorMessage) {
	for _, rumor := range g.KnownRumors {
		if strings.Compare(rumor.Origin, r.Origin) == 0 && rumor.ID == r.ID {
			return
		}
	}
	g.KnownRumors = append(g.KnownRumors, r)
}

// Update Want slice for given origin
func updateWant(g *core.Gossiper, origin string) {
	if strings.Compare(origin, "") == 0 {
		return
	}
	for i, peerStatus := range g.Want {
		if strings.Compare(peerStatus.Identifier, origin) == 0 {
			g.Want[i].NextID++
			return
		}
	}
	// Create a new PeerStatus if none exists
	newPeerStatus := core.PeerStatus{
		Identifier: origin,
		NextID:     uint32(2),
	}
	g.Want = append(g.Want, newPeerStatus)
}

// Get the gossiper current ID from its own Rumors. Useful when reconnecting
// to the network after having already sent some Rumors
func adjustMyCurrentID(g *core.Gossiper, status core.StatusPacket) {
	g.RumorIDLock.Lock()
	if g.CurrentRumorID == uint32(0) {
		currentMaxIDFromRumors := uint32(0)
		for _, st := range status.Want {
			if strings.Compare(st.Identifier, g.Name) == 0 && st.NextID > currentMaxIDFromRumors {
				currentMaxIDFromRumors = st.NextID
			}
		}
		if currentMaxIDFromRumors > 0 {
			g.CurrentRumorID = currentMaxIDFromRumors - 1
		}
	}
	g.RumorIDLock.Unlock()
}

// CreateSliceKnownPeers Create the slice of known peers necessary given a string containing
// all addresses
func CreateSliceKnownPeers(knownPeers string) []string {
	list := make([]string, 0)

	// No known-peers case
	if strings.Compare(knownPeers, "") == 0 {
		return list
	}

	// Fill the list
	s := ""
	for _, c := range knownPeers {
		if string(c) != "," {
			s += string(c)
		} else {
			list = append(list, s)
			s = ""
		}
	}
	list = append(list, s)

	return list
}
