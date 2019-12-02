package gossiper

import (
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/AleksandarHrusanov/Peerster/core"
	"github.com/AleksandarHrusanov/Peerster/filehandling"
	"github.com/AleksandarHrusanov/Peerster/helpers"
	"github.com/dedis/protobuf"
)

// TODO: Break this function into shorter separate functions
// Main peersListener function
func peersListener(gossiper *core.Gossiper, simpleMode bool) {
	// Remove all timed-out or handled mongering statuses
	go mongeringStatusRefresher(gossiper)

	// Infinite loop
	for {
		var gossipPacket core.GossipPacket
		var fromAddrPtr *net.UDPAddr
		fromAddr := ""

		// Listen
		gossipPacket, fromAddrPtr = receiveAndDecode(gossiper)
		if fromAddrPtr != nil {
			fromAddr = fromAddrPtr.String()
		}

		// Store address from the sender
		gossiper.PeersLock.Lock()
		knownPeers := gossiper.KnownPeers
		if !helpers.SliceContainsString(knownPeers, fromAddr) &&
			strings.Compare(fromAddr, "") != 0 {
			gossiper.KnownPeers = append(knownPeers, fromAddr)
		}
		gossiper.PeersLock.Unlock()

		if gossipPacket.Simple != nil {
			// Print simple output
			simpleMessage := *gossipPacket.Simple
			// helpers.PrintOutputSimpleMessageFromPeer(simpleMessage.Contents,
			// 	simpleMessage.OriginalName,
			// 	simpleMessage.RelayPeerAddr,
			// 	gossiper.KnownPeers)

			if simpleMode {
				// Prepare the message to be sent (SIMPLE MODE)
				simpleMessage.RelayPeerAddr = gossiper.Address.String()
				packetToSend := core.GossipPacket{Simple: &simpleMessage}
				packetBytes, err := protobuf.Encode(&packetToSend)
				helpers.HandleErrorFatal(err)

				// Send message to all other known peers
				for _, knownAddress := range knownPeers {
					if strings.Compare(knownAddress, fromAddr) != 0 {
						core.ConnectAndSend(knownAddress, gossiper.Conn, packetBytes)
					}
				}
			}
		}

		if !simpleMode {
			gossiper.PeersLock.Lock()
			knownPeers := gossiper.KnownPeers
			gossiper.PeersLock.Unlock()
			if gossipPacket.DataRequest != nil {
				// Handle incoming data requests messages from other peers
				filehandling.HandlePeerDataRequest(gossiper, gossipPacket.DataRequest)
			} else if gossipPacket.DataReply != nil {
				// Handle incoming data reply messages from other peers
				filehandling.HandlePeerDataReply(gossiper, gossipPacket.DataReply)
			} else if gossipPacket.SearchRequest != nil {
				filehandling.HandlePeerSearchRequest(gossiper, gossipPacket.SearchRequest)
			} else if gossipPacket.SearchReply != nil {
				filehandling.HandlePeerSearchReply(gossiper, gossipPacket.SearchReply)
			} else if gossipPacket.Private != nil {
				// Handle incoming private message from another peer
				handlePrivateMessage(gossiper, gossipPacket.Private)
			} else if gossipPacket.Rumor != nil {
				// Print RumorFromPeer output
				// helpers.PrintOutputRumorFromPeer(gossipPacket.Rumor.Origin, fromAddr, gossipPacket.Rumor.ID, gossipPacket.Rumor.Text, knownPeers)

				// Check if the Rumor or its Origin is known
				rumorIsKnown, originIsKnown, wantedID := core.IsRumorKnown(gossiper.Want, gossipPacket.Rumor)

				if rumorIsKnown {
					// Do nothing

				} else if originIsKnown && wantedID == gossipPacket.Rumor.ID {
					// Update wantedID in Want slice
					updateWant(gossiper, gossipPacket.Rumor.Origin)

				} else if originIsKnown && wantedID < gossipPacket.Rumor.ID {
					// Do nothing

				} else if !originIsKnown {
					if gossipPacket.Rumor.ID == 1 {
						// If ID = 1, new Rumor: add it to list of known rumors
						// and create new PeerStatus
						updateWant(gossiper, gossipPacket.Rumor.Origin)

					} else {
						// If ID > 1, create new PeerStatus
						newPeerStatus := core.PeerStatus{
							Identifier: gossipPacket.Rumor.Origin,
							NextID:     uint32(1),
						}
						gossiper.Want = append(gossiper.Want, newPeerStatus)
					}
				}

				// Update destiantionTable
				gossiper.DestinationTable.DsdvLock.Lock()
				core.UpdateDestinationTable(gossiper.Name, gossipPacket.Rumor.Origin, gossipPacket.Rumor.ID, fromAddr,
					gossiper.DestinationTable.Dsdv, gossiper.KnownRumors, originIsKnown, !core.IsRouteRumor(gossipPacket.Rumor))
				gossiper.DestinationTable.DsdvLock.Unlock()

				// Send status
				sendStatus(gossiper, fromAddr)

				if !rumorIsKnown {
					if len(knownPeers) > 0 {
						// Pick a random address and send the rumor
						chosenAddr := helpers.PickRandomInSlice(knownPeers)
						sendRumor(*gossipPacket.Rumor, gossiper, chosenAddr)
						// helpers.PrintOutputMongering(chosenAddr)
					}
				}

				// Add Rumor to list of known Rumors if it is not already there
				addRumorToKnownRumors(gossiper, *gossipPacket.Rumor)
			} else if gossipPacket.Status != nil {
				// Print STATUS message
				// core.PrintOutputStatus(fromAddr, gossipPacket.Status.Want, gossiper.KnownPeers)

				// Check own rumorID to avoid crashes after reconnection (TODO)
				adjustMyCurrentID(gossiper, *gossipPacket.Status)

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
								for _, want := range gossipPacket.Status.Want {
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

				for _, peerStatus := range gossipPacket.Status.Want {
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
						for _, peerStatus := range gossipPacket.Status.Want {
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
		}
	}
}

func clientListener(gossiper *core.Gossiper, simpleMode bool) {
	for {
		gossiper.PeersLock.Lock()
		knownPeers := gossiper.KnownPeers
		gossiper.PeersLock.Unlock()
		// Receive and decode messages
		message, _ := receiveAndDecodeFromClient(gossiper)

		if simpleMode {
			// Print simple output
			// helpers.PrintOutputSimpleMessageFromClient(message.Text, gossiper.KnownPeers)

			// Prepare the message to be sent (SIMPLE MODE)
			simpleMessageToSend := core.SimpleMessage{
				OriginalName:  gossiper.Name,
				RelayPeerAddr: gossiper.Address.String(),
				Contents:      message.Text,
			}
			packetToSend := core.GossipPacket{Simple: &simpleMessageToSend}
			packetBytes, err := protobuf.Encode(&packetToSend)
			helpers.HandleErrorFatal(err)

			// Send message to all known peers
			for _, knownAddress := range knownPeers {
				core.ConnectAndSend(knownAddress, gossiper.Conn, packetBytes)
			}
		}

		// Prepare the message to be sent
		if !simpleMode {
			if isClientFileIndexing(&message) {
				//Handle messages from client to simply index a file
				filehandling.HandleFileIndexing(gossiper, *message.File)
			} else if isClientRequestingDownload(&message) {
				// Handle message from client to request a file download
				filehandling.HandleClientDownloadRequest(gossiper, &message)
			} else if isClientRequestingFileSearch(&message) {
				filehandling.HandleClientSearchRequest(gossiper, &message)
			} else if isClientRequestingImplicitDownload(&message) {
				filehandling.HandleClientImplicitDownloadRequest(gossiper, &message)
			} else {
				if isClientMessagePrivate(&message) {
					// Handle private messages from client
					privateMsg := createNewPrivateMessage(gossiper.Name, message.Text, message.Destination)
					handlePrivateMessage(gossiper, privateMsg)
				} else {
					// Print output
					// helpers.PrintOutputSimpleMessageFromClient(message.Text, gossiper.KnownPeers)

					// Add rumor to list of known rumors
					gossiper.RumorIDLock.Lock()
					gossiper.CurrentRumorID++
					newRumor := core.RumorMessage{
						Origin: gossiper.Name,
						ID:     gossiper.CurrentRumorID,
						Text:   message.Text,
					}
					gossiper.RumorIDLock.Unlock()
					addRumorToKnownRumors(gossiper, newRumor)
					updateWant(gossiper, gossiper.Name)

					// Pick a random address and send the rumor
					chosenAddr := ""
					if len(knownPeers) > 0 {
						chosenAddr = helpers.PickRandomInSlice(knownPeers)
						sendRumor(newRumor, gossiper, chosenAddr)
						// helpers.PrintOutputMongering(chosenAddr)
					}
				}
			}
		}
	}
}

// =====================================================================
// =====================================================================
//													Utils
// =====================================================================
// =====================================================================

// true if the client did not specify a destination - only wants to index and divide file locally
func isClientFileIndexing(clientMsg *core.Message) bool {
	return (strings.Compare(*(clientMsg.File), "") != 0 &&
		(strings.Compare(*(clientMsg.Destination), "") == 0) &&
		len(*clientMsg.Request) == 0)
}

func isClientRequestingFileSearch(clientMsg *core.Message) bool {
	return (strings.Compare(*(clientMsg.Keywords), "") != 0)
}

// true if the client did not specify a destination - only wants to index and divide file locally
func isClientRequestingDownload(clientMsg *core.Message) bool {
	return (strings.Compare(*(clientMsg.File), "") != 0 &&
		(strings.Compare(*(clientMsg.Destination), "") != 0) &&
		(len(*clientMsg.Request) != 0))
}

// true if the client did not specify a destination - only wants to index and divide file locally
func isClientRequestingImplicitDownload(clientMsg *core.Message) bool {
	return (strings.Compare(*(clientMsg.File), "") != 0 &&
		(strings.Compare(*(clientMsg.Destination), "") == 0) &&
		(len(*clientMsg.Request) != 0))
}
