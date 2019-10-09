package gossiper

import "github.com/JohnDoe/Peerster/structs"
import "github.com/JohnDoe/Peerster/helpers"

// This file contains functions to handle each type of packets (simple, rumor, and status)

func handleIncomingSimplePacket(gossiper *structs.Gossiper, packet *structs.GossipPacket, senderAddr string) {
  gossiper.Peers[senderAddr] = true
  helpers.WriteToStandardOutputWhenPeerSimpleMessageReceived(gossiper, packet)
  packet.Simple.RelayPeerAddr = gossiper.Address.String()
  broadcastGossipPacket(gossiper, packet, senderAddr)
}

func handleIncomingRumorPacket(gossiper *structs.Gossiper, packet *structs.GossipPacket, senderAddr string) {

}

func handleIncomingStatusPacket(gossiper *structs.Gossiper, packet *structs.GossipPacket, senderAddr string) {

}
