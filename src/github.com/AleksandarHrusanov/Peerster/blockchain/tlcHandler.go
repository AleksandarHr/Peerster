package blockchain

import (
	"encoding/hex"
	"math/rand"
	"strings"
	"time"

	"github.com/AleksandarHrusanov/Peerster/core"
	"github.com/AleksandarHrusanov/Peerster/helpers"
	"github.com/dedis/protobuf"
)

func HandleTLCMessage(gossiper *core.Gossiper, tlc *core.TLCMessage, peerCount int, ackHopLimit uint32, fromAddr string) {
	// add TLC to knownTLCs if it is new
	// TODO: Check if txn is valid
	alreadySeen := addOrUpdateKnownTLC(gossiper, tlc)
	if tlc.Confirmed == -1 {
		// if receiving an unconfirmed tlc message
		helpers.PrintUnconfirmedGossip(tlc.Origin, tlc.TxBlock.Transaction.Name,
			hex.EncodeToString(tlc.TxBlock.Transaction.MetafileHash), tlc.ID, tlc.TxBlock.Transaction.Size)

		packetToSend := core.GossipPacket{TLCMessage: tlc}
		packetBytes, err := protobuf.Encode(&packetToSend)
		helpers.HandleErrorFatal(err)

		gossiper.PeersLock.Lock()
		knownPeers := gossiper.KnownPeers
		gossiper.PeersLock.Unlock()

		if alreadySeen {
			// flip a coin to decide whether to send to a peer or not
			rand.Seed(time.Now().UnixNano())
			res := rand.Intn(2)
			if res == 1 {
				// send to a random peer
				if len(knownPeers) > 0 {
					chosenAddr := helpers.PickRandomInSliceDifferentFrom(knownPeers, fromAddr)
					core.ConnectAndSend(chosenAddr, gossiper.Conn, packetBytes)
				}
			}
		} else {
			// send to a random peer
			if len(knownPeers) > 0 {
				chosenAddr := helpers.PickRandomInSliceDifferentFrom(knownPeers, fromAddr)
				core.ConnectAndSend(chosenAddr, gossiper.Conn, packetBytes)
			}

			// create a TLCAck
			ack := &core.TLCAck{Origin: gossiper.Name, ID: tlc.ID, Text: "", Destination: tlc.Origin, HopLimit: ackHopLimit}

			// Send the TLCAck
			helpers.PrintSendingAck(ack.Origin, ack.ID)
			HandleTlcAck(gossiper, ack, peerCount)
		}
	} else {
		// receiving a confirmed tlc message
		helpers.PrintConfirmedGossip(tlc.Origin, tlc.TxBlock.Transaction.Name,
			hex.EncodeToString(tlc.TxBlock.Transaction.MetafileHash), tlc.ID, tlc.TxBlock.Transaction.Size)
		if alreadySeen {
			// update if already seen, otherwise do nothing
			updateTLC(gossiper, tlc)
		}
	}
}

// Add a TLC to the gossiper's known TLCs if it is not already there
func addOrUpdateKnownTLC(gossiper *core.Gossiper, t *core.TLCMessage) bool {
	gossiper.TLCLock.Lock()
	for idx, tlc := range gossiper.KnownTLCs {
		if strings.Compare(tlc.Origin, t.Origin) == 0 && tlc.ID == t.ID {
			gossiper.KnownTLCs[idx] = *t
			gossiper.TLCLock.Unlock()
			return true
		}
	}
	gossiper.KnownTLCs = append(gossiper.KnownTLCs, *t)
	gossiper.TLCLock.Unlock()
	return false
}

// adds a tlc message to the OwnTLC struct of gossiper
func createAndAddOwnTLC(gossiper *core.Gossiper, t *core.TLCMessage) {
	gossiper.TLCLock.Lock()
	witnesses := make(map[string]bool, 0)
	witnesses[gossiper.Name] = true
	ownTlc := core.OwnTLC{TLC: *t, AcksReceived: 1, Witnesses: witnesses}
	gossiper.MyTLCs[t.ID] = ownTlc
	gossiper.TLCLock.Unlock()
}

// updates a tlc message in the gossiper's KnownTLC sturct
func updateTLC(g *core.Gossiper, t *core.TLCMessage) {
	for idx, tlc := range g.KnownTLCs {
		if strings.Compare(tlc.Origin, t.Origin) == 0 && tlc.ID == t.ID {
			g.KnownTLCs[idx] = *t
		}
	}
}

// Handles a received TLC Ack
func HandleTlcAck(gossiper *core.Gossiper, ack *core.TLCAck, peerCount int) {
	if strings.Compare(gossiper.Name, ack.Destination) != 0 {
		// If ack is not for this gossiper, simply forward with next hop
		forwardTlcAck(gossiper, ack)
	} else {
		// if ack is for this gossiper, increment ack count
		confirmed := updateTlcOnReceivedAck(gossiper, ack, peerCount)
		if confirmed {
			// if TLC message is now confirmed (e.g. has majority acks)
			gossiper.TLCLock.Lock()
			ownTlc := gossiper.MyTLCs[ack.ID]
			confirmedTlc := ownTlc.TLC

			// Assign confirmed TLC's Confirmed field to the original TLC message ID
			shouldResend := confirmedTlc.Confirmed == -1
			confirmedTlc.Confirmed = int(confirmedTlc.ID)

			// Assign confirmed TLC's ID to next available mongering ID
			gossiper.CurrentMongeringID++
			confirmedTlc.ID = gossiper.CurrentMongeringID

			ownTlc.TLC = confirmedTlc
			gossiper.MyTLCs[ack.ID] = ownTlc
			updateTLC(gossiper, &confirmedTlc)
			gossiper.TLCLock.Unlock()

			packetToSend := core.GossipPacket{TLCMessage: &confirmedTlc}
			packetBytes, err := protobuf.Encode(&packetToSend)
			helpers.HandleErrorFatal(err)

			gossiper.PeersLock.Lock()
			knownPeers := gossiper.KnownPeers
			gossiper.PeersLock.Unlock()

			// send to a random peer
			if len(knownPeers) > 0 && shouldResend {
				chosenAddr := helpers.PickRandomInSlice(knownPeers)
				helpers.PrintReBroadcastId(confirmedTlc.ID, ownTlc.Witnesses)
				core.ConnectAndSend(chosenAddr, gossiper.Conn, packetBytes)
			}
		}
	}
}

// Updates gossiper's OwnTLC struct on receiving an ack
func updateTlcOnReceivedAck(gossiper *core.Gossiper, ack *core.TLCAck, peerCount int) bool {
	gossiper.TLCLock.Lock()
	ackedTlc := gossiper.MyTLCs[ack.ID]
	ackedTlc.AcksReceived++
	confirmed := (peerCount/2 + 1) <= ackedTlc.AcksReceived
	ackedTlc.Witnesses[ack.Origin] = true
	gossiper.MyTLCs[ack.ID] = ackedTlc
	gossiper.TLCLock.Unlock()

	return confirmed
}

// Sends TLC message every stubbornTimeout seconds until it has been ack'ed by majority
func StubbornlySendTLC(gossiper *core.Gossiper, newFile *core.FileInformation, stubbornTimeout int) {
	// Create and add new TLC to knownTLCs
	blockPublish := CreateBlockPublish(newFile.FileName, newFile.Size, newFile.MetaHash[:])
	newTLC := CreateTLCMessage(gossiper, *blockPublish)
	addOrUpdateKnownTLC(gossiper, newTLC)
	createAndAddOwnTLC(gossiper, newTLC)
	gossiper.PeersLock.Lock()
	knownPeers := gossiper.KnownPeers
	gossiper.PeersLock.Unlock()

	for {
		gossiper.TLCLock.Lock()
		ownTlc := gossiper.MyTLCs[newTLC.ID]
		updatedTlc := ownTlc.TLC
		gossiper.TLCLock.Unlock()

		if updatedTlc.Confirmed == -1 {
			// If stubbornTimeout passed and TLC message is still unconfirmed
			// simply send to a random peer
			packetToSend := core.GossipPacket{TLCMessage: &updatedTlc}
			packetBytes, err := protobuf.Encode(&packetToSend)
			helpers.HandleErrorFatal(err)
			chosenAddr := ""
			if len(knownPeers) > 0 {
				chosenAddr = helpers.PickRandomInSlice(knownPeers)
				core.ConnectAndSend(chosenAddr, gossiper.Conn, packetBytes)
				helpers.PrintUnconfirmedGossip(updatedTlc.Origin, updatedTlc.TxBlock.Transaction.Name,
					hex.EncodeToString(updatedTlc.TxBlock.Transaction.MetafileHash), updatedTlc.ID, updatedTlc.TxBlock.Transaction.Size)
			}

			// wait for stubbornTimeout seconds
			time.Sleep(time.Duration(stubbornTimeout) * time.Second)
		} else {
			return
		}
	}
}

// A function to forward a TLCAck to the corresponding next hop
func forwardTlcAck(gossiperPtr *core.Gossiper, ack *core.TLCAck) {

	if ack.HopLimit == 0 {
		// if we have reached the HopLimit, drop the message
		return
	}

	gossiperPtr.DestinationTable.DsdvLock.Lock()
	forwardingAddress := gossiperPtr.DestinationTable.Dsdv[ack.Destination]
	gossiperPtr.DestinationTable.DsdvLock.Unlock()
	// If current node has no information about next hop to the destination in question
	if strings.Compare(forwardingAddress, "") == 0 {
		// TODO: What to do if there is no 'next hop' known when peer has to forward a private packet
	}

	// Decrement the HopLimit right before forwarding the packet
	ack.HopLimit--
	// Encode and send packet
	packetToSend := core.GossipPacket{Ack: ack}
	packetBytes, err := protobuf.Encode(&packetToSend)
	helpers.HandleErrorFatal(err)
	core.ConnectAndSend(forwardingAddress, gossiperPtr.Conn, packetBytes)
}
