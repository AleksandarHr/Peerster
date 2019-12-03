package gossiper

import (
	"math/rand"
	"strings"
	"time"

	"github.com/AleksandarHrusanov/Peerster/core"
	"github.com/AleksandarHrusanov/Peerster/helpers"
)

// Repeat the rumor mongering process
func rumorMongerAgain(g *core.Gossiper) {
	// Send again the rumor to a different address
	rumorToSend := g.MongeringStatus[0].RumorMessage
	chosenAddr := helpers.PickRandomInSliceDifferentFrom(g.KnownPeers,
		g.MongeringStatus[0].WaitingStatusFromAddr)
	if strings.Compare(chosenAddr, "") != 0 {
		safeMongeringStatusDelete(g)
		sendRumor(rumorToSend, g, chosenAddr)
		helpers.PrintOutputMongering(chosenAddr)
	} else {
		safeMongeringStatusDelete(g)
	}
}

// Safe delete of mongering status
func safeMongeringStatusDelete(g *core.Gossiper) {
	statusToDelete := g.MongeringStatus[0]
	statusToDelete.Lock.Lock()
	g.MongeringStatus = g.MongeringStatus[1:]
	statusToDelete.Lock.Unlock()
}

// Remove all mongering status that timed-out and repeat the mongering process
func mongeringStatusRefresher(g *core.Gossiper) {
	for {
		if len(g.MongeringStatus) > 0 {
			if g.MongeringStatus[0] != nil {
				select {
				case <-g.MongeringStatus[0].TimeUp:
					// Timed-out
					rumorMongerAgain(g)
				default:
					// Already acknowledged
					if g.MongeringStatus[0].AckReceived {
						safeMongeringStatusDelete(g)
					}
				}
			}
			// Wait
			time.Sleep(5 * time.Millisecond)
		} else {
			// Wait
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func handleRumorMessage(gossiper *core.Gossiper, rumor *core.RumorMessage, fromAddr string, knownPeers []string) {
	// Check if the Rumor or its Origin is known
	rumorIsKnown, originIsKnown, wantedID := core.IsRumorKnown(gossiper.Want, rumor)

	if rumorIsKnown {
		// Do nothing

	} else if originIsKnown && wantedID == rumor.ID {
		// Update wantedID in Want slice
		updateWant(gossiper, rumor.Origin)

	} else if originIsKnown && wantedID < rumor.ID {
		// Do nothing

	} else if !originIsKnown {
		if rumor.ID == 1 {
			// If ID = 1, new Rumor: add it to list of known rumors
			// and create new PeerStatus
			updateWant(gossiper, rumor.Origin)

		} else {
			// If ID > 1, create new PeerStatus
			newPeerStatus := core.PeerStatus{
				Identifier: rumor.Origin,
				NextID:     uint32(1),
			}
			gossiper.Want = append(gossiper.Want, newPeerStatus)
		}
	}

	// Update destiantionTable
	gossiper.DestinationTable.DsdvLock.Lock()
	core.UpdateDestinationTable(gossiper.Name, rumor.Origin, rumor.ID, fromAddr,
		gossiper.DestinationTable.Dsdv, gossiper.KnownRumors, originIsKnown, !core.IsRouteRumor(rumor))
	gossiper.DestinationTable.DsdvLock.Unlock()

	// Send status
	sendStatus(gossiper, fromAddr)

	if !rumorIsKnown {
		if len(knownPeers) > 0 {
			// Pick a random address and send the rumor
			chosenAddr := helpers.PickRandomInSlice(knownPeers)
			sendRumor(*rumor, gossiper, chosenAddr)
			// helpers.PrintOutputMongering(chosenAddr)
		}
	}

	// Add Rumor to list of known Rumors if it is not already there
	addRumorToKnownRumors(gossiper, *rumor)
}

func handleStatusPacket(gossiper *core.Gossiper, statusPckt *core.StatusPacket, fromAddr string, knownPeers []string) {
	// Check own rumorID to avoid crashes after reconnection (TODO)
	adjustMyCurrentID(gossiper, *statusPckt)

	// Check if the gossiper was waiting for this status packet and retrieve the
	// corresponding Rumor if we need to send it again after a coin flip
	rumorsToFlipCoinFor := make([]core.RumorMessage, 0)
	// For each mongeringstatus
	for _, mongeringStatus := range gossiper.MongeringStatus {
		// Check if the mongeringStatus has not yet been deleted
		if mongeringStatus != nil {
			mongeringStatus.Lock.Lock()
		} else {
			continue
		}
		// Check that we were waiting an answer from this address
		if strings.Compare(mongeringStatus.WaitingStatusFromAddr, fromAddr) == 0 {
			select {
			case _, ok := <-mongeringStatus.TimeUp:
				// Do nothing if it has timed-up
				_ = ok
			default:
				// Check which Rumors have been acknowledge if any, can acknowldge
				// more than one Rumor
				if !mongeringStatus.AckReceived {
					// For each want of the other gossiper
					for _, want := range statusPckt.Want {
						if strings.Compare(want.Identifier, mongeringStatus.RumorMessage.Origin) == 0 &&
							want.NextID > mongeringStatus.RumorMessage.ID {
							rumorsToFlipCoinFor = updateRumorListNoDuplicates(mongeringStatus.RumorMessage,
								rumorsToFlipCoinFor)
							mongeringStatus.AckReceived = true
						}
					}
				}
			}
		}
		mongeringStatus.Lock.Unlock()
	}

	// Check the received status packet
	iWantYourRumors := false
	youWantMyRumors := false
	var peerStatusTemp core.PeerStatus

	for _, peerStatus := range statusPckt.Want {
		peerFound := false
		for _, ownPeerStatus := range gossiper.Want {
			if strings.Compare(peerStatus.Identifier, ownPeerStatus.Identifier) == 0 {
				peerFound = true
				if peerStatus.NextID < ownPeerStatus.NextID {
					// The other peer has not yet seen some of the Rumor I have
					youWantMyRumors = true
					peerStatusTemp = peerStatus
				} else if peerStatus.NextID > ownPeerStatus.NextID {
					// The other peer has some Rumor I do not have
					iWantYourRumors = true
				}
			}
		}
		// Case: the peer know a peer I do not know
		if !peerFound {
			iWantYourRumors = true
			newPeerStatus := core.PeerStatus{
				Identifier: peerStatus.Identifier,
				NextID:     1,
			}
			gossiper.Want = append(gossiper.Want, newPeerStatus)
		}
	}

	// Check if the other peer know the same peer as I do
	withFreshID := false
	if !iWantYourRumors && !youWantMyRumors {
		for _, ownPeerStatus := range gossiper.Want {
			peerFound := false
			for _, peerStatus := range statusPckt.Want {
				if strings.Compare(peerStatus.Identifier, ownPeerStatus.Identifier) == 0 {
					peerFound = true
				}
			}
			// Case: I know a peer that the other peer don't
			if !peerFound {
				youWantMyRumors = true
				peerStatusTemp = ownPeerStatus
				withFreshID = true
			}
		}
	}

	// Solve the cases
	if youWantMyRumors {
		var rumorToSend *core.RumorMessage
		if withFreshID {
			rumorToSend = getRumor(gossiper, peerStatusTemp.Identifier, 1)
		} else {
			rumorToSend = getRumor(gossiper, peerStatusTemp.Identifier, peerStatusTemp.NextID)
		}
		if rumorToSend != nil {
			// helpers.PrintOutputMongering(fromAddr)
			sendRumor(*rumorToSend, gossiper, fromAddr)
		}
	} else if iWantYourRumors {
		sendStatus(gossiper, fromAddr)
	} else {
		// Case: each peer is up-to-date and the gossiper
		// decide to continue rumormongering or not
		// Print IN SYNC WITH message
		// helpers.PrintOutputInSyncWith(fromAddr)
		// Continue mongering test
		if len(rumorsToFlipCoinFor) > 0 {
			for _, rToFlip := range rumorsToFlipCoinFor {
				seed := rand.NewSource(time.Now().UnixNano())
				rng := rand.New(seed)
				flipCoinResult := rng.Intn(2)
				if flipCoinResult == 0 {
					// Pick a random address and send the rumor
					chosenAddr := helpers.PickRandomInSliceDifferentFrom(knownPeers, fromAddr)
					if strings.Compare(chosenAddr, "") != 0 {
						// Print FLIPPED COIN message and send rumor
						sendRumor(rToFlip, gossiper, chosenAddr)
						// helpers.PrintOutputFlippedCoin(chosenAddr)
					}
				}
			}
		}
	}
}
