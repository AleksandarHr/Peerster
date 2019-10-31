package gossiper

import (
	"strings"
	"time"

	"github.com/2_alt/Peerster/core"
	"github.com/2_alt/Peerster/helpers"
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
