package gossiper

import (
	"encoding/hex"
	"net"
	"strings"
	"time"

	"github.com/AleksandarHrusanov/Peerster/blockchain"
	"github.com/AleksandarHrusanov/Peerster/core"
	"github.com/AleksandarHrusanov/Peerster/filehandling"
	"github.com/AleksandarHrusanov/Peerster/helpers"
	"github.com/dedis/protobuf"
)

// TODO: Break this function into shorter separate functions
// Main peersListener function
func peersListener(gossiper *core.Gossiper, simpleMode bool, peerCount int) {
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
			} else if gossipPacket.TLCMessage != nil {
				blockchain.HandleTLCMessage(gossiper, gossipPacket.TLCMessage, peerCount)
			} else if gossipPacket.Ack != nil {
				blockchain.HandleTlcAck(gossiper, gossipPacket.Ack, peerCount)
			} else if gossipPacket.Rumor != nil {
				// Print RumorFromPeer output
				// helpers.PrintOutputRumorFromPeer(gossipPacket.Rumor.Origin, fromAddr, gossipPacket.Rumor.ID, gossipPacket.Rumor.Text, knownPeers)
				handleRumorMessage(gossiper, gossipPacket.Rumor, fromAddr, knownPeers)
			} else if gossipPacket.Status != nil {
				// Print STATUS message
				// core.PrintOutputStatus(fromAddr, gossipPacket.Status.Want, gossiper.KnownPeers)
				handleStatusPacket(gossiper, gossipPacket.Status, fromAddr, knownPeers)
			}
		}
	}
}

func clientListener(gossiper *core.Gossiper, simpleMode bool, stubbornTimeout int) {
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
				newFile := filehandling.HandleFileIndexing(gossiper, *message.File)

				go func(gossiper *core.Gossiper, newFile *core.FileInformation) {
					// Create and add new TLC to knownTLCs
					blockPublish := blockchain.CreateBlockPublish(newFile.FileName, newFile.Size, newFile.MetaHash[:])
					newTLC := blockchain.CreateTLCMessage(gossiper, *blockPublish)
					blockchain.AddTLCToKnownTLCs(gossiper, newTLC)
					blockchain.AddOwnTLC(gossiper, newTLC)

					for {
						ownTlc := gossiper.MyTLCs[newTLC.ID]
						updatedTlc := ownTlc.TLC
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
								helpers.PrintUnconfirmedGossip(gossiper.Name, newFile.FileName,
									hex.EncodeToString(newFile.MetaHash[:]), updatedTlc.ID, updatedTlc.TxBlock.Transaction.Size)
							}
							time.Sleep(time.Duration(stubbornTimeout) * time.Second)
						} else {
							break
						}
					}
				}(gossiper, newFile)

				// monger tlc message just like a rumor message
				// updateWant(gossiper, gossiper.Name)
				//
				// // Pick a random address and send the tlc message
				// chosenAddr := ""
				// if len(knownPeers) > 0 {
				// 	chosenAddr = helpers.PickRandomInSlice(knownPeers)
				// 	sendTLC(newTLC, gossiper, chosenAddr)
				// 	helpers.PrintUnconfirmedGossip(newTLC.Origin, newFile.FileName, hex.EncodeToString(newFile.MetaHash[:]), newTLC.ID, blockPublish.Transaction.Size)
				// }
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
					gossiper.MongeringIDLock.Lock()
					gossiper.CurrentMongeringID++
					newRumor := core.RumorMessage{
						Origin: gossiper.Name,
						ID:     gossiper.CurrentMongeringID,
						Text:   message.Text,
					}
					gossiper.MongeringIDLock.Unlock()
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
