package core

import (
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/AleksandarHrusanov/Peerster/helpers"
)

//===============================================
//===============================================
// 							Gossiper Getters
//===============================================
//===============================================

// GetAllSharedFilesAndHashes - a function to return names and metahashes of all shared files
func (g *Gossiper) GetAllSharedFilesAndHashes() map[string]string {
	return g.FilesAndMetahashes.FileNamesToMetahashesMap
}

// GetUIPort Get the gossiper's UI port
func (g *Gossiper) GetUIPort() string {
	return g.uiPort
}

// GetLocalAddr Get the gossiper's localConn as a string
func (g *Gossiper) GetLocalAddr() string {
	return g.LocalAddr.String()
}

// GetAllRumors Get the rumors known by the gossiper
func (g *Gossiper) GetAllRumors() []RumorMessage {
	return g.KnownRumors
}

// GetAllRumors Get the rumors known by the gossiper
func (g *Gossiper) GetAllFullyMatchedFilenames() []string {
	return g.OngoingFileSearch.MatchesFileNames
}

func (g *Gossiper) GetMetafileHashByName(fname string) string {
	finfo := g.OngoingFileSearch.MatchesFound[fname]
	return hex.EncodeToString(finfo.Metahash)
}

// GetAllFileNames - return an array of filenames shared/downloaded by the gossiper
func (g *Gossiper) GetAllFileNames() []string {
	g.FilesAndMetahashes.FilesLock.Lock()
	filenames := make([]string, 0)

	for fname, _ := range g.FilesAndMetahashes.FileNamesToMetahashesMap {
		filenames = append(filenames, fname)
	}
	g.FilesAndMetahashes.FilesLock.Unlock()

	return filenames
}

func (g *Gossiper) GetAllKnownFiles() *SafeFilesAndMetahashes {
	g.FilesAndMetahashes.FilesLock.Lock()
	files := g.FilesAndMetahashes
	g.FilesAndMetahashes.FilesLock.Unlock()
	return files
}

// GetAllNonRouteRumors Get the rumors known by the gossiper
func (g *Gossiper) GetAllNonRouteRumors() []RumorMessage {
	allRumors := g.KnownRumors
	regularRumors := make([]RumorMessage, 0)
	for _, r := range allRumors {
		if strings.Compare(r.Text, "") != 0 {
			regularRumors = append(regularRumors, r)
		}
	}
	return regularRumors
}

// GetAllKnownPeers Get the known peers of this gossiper
func (g *Gossiper) GetAllKnownPeers() []string {
	return g.KnownPeers
}

// GetAllPrivateMessagesBetween Get the known peers of this gossiper
func (g *Gossiper) GetAllPrivateMessagesBetween() map[string][]string {
	return g.PrivateMessages.Messages
}

// GetAllKnownOrigins - returns the origins known to this gossiper
func (g *Gossiper) GetAllKnownOrigins() []string {
	origins := make([]string, 0)
	g.DestinationTable.DsdvLock.Lock()
	for o := range g.DestinationTable.Dsdv {
		origins = append(origins, o)
	}
	g.DestinationTable.DsdvLock.Unlock()

	sort.Strings(origins)
	return origins
}

// AddPeer Add a peer to the list of known peers
func (g *Gossiper) AddPeer(address string) {
	if helpers.IPAddressIsValid(address) {
		for _, peer := range g.KnownPeers {
			if strings.Compare(peer, address) == 0 {
				return
			}
		}
		g.KnownPeers = append(g.KnownPeers, address)
	}
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
func UpdateDestinationTable(ownName string, rumorOrigin string, rumorID uint32, fromAddr string,
	destinationTable map[string]string, knownRumors []RumorMessage, originIsKnown bool, toPrint bool) {

	toPrint = true
	if strings.Compare(ownName, rumorOrigin) != 0 {
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
				if toPrint {
					helpers.PrintOutputUpdatingDSDV(rumorOrigin, fromAddr)
				}
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
