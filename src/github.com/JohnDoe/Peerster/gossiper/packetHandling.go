package gossiper

import "fmt"
import "github.com/JohnDoe/Peerster/structs"
import "github.com/JohnDoe/Peerster/helpers"

// This file contains functions to handle each type of packets (simple, rumor, and status)

func handleIncomingSimplePacket(gossiper *structs.Gossiper, packet *structs.GossipPacket, senderAddr string) {
  helpers.WriteToStandardOutputWhenPeerSimpleMessageReceived(gossiper, packet)
  packet.Simple.RelayPeerAddr = gossiper.Address.String()
  broadcastGossipPacket(gossiper, packet, senderAddr)
}

func handleIncomingRumorPacket(gossiper *structs.Gossiper, packet *structs.GossipPacket, senderAddr string) {

  rumor := packet.Rumor
  if rumor == nil {
    fmt.Println("Received an rumor packet with no rumor in it.")
    return
  }

  // If message is new, print to standard output, update seen messages and chanel map,
  if !helpers.AlreadySeenMessage(gossiper, rumor) {
    fmt.Println("A new message has been received - adding to SEEN")
    helpers.WriteToStandardOutputWhenRumorMessageReceived(gossiper, packet, senderAddr)
    // If this is the first rumor packet of a rumormongering session, update the map of chanels of this node
    gossiper.MapHandler <- senderAddr
    updateSeenMessages(gossiper, rumor)
  }

  // Update PeerStatus information
  fmt.Println("Updating known peers")
  peerStatus := structs.CreateNewPeerStatusPair(rumor.Origin, uint32(rumor.ID + 1))
  updatePeerStatusList(gossiper, peerStatus)

  // Send status packet
  fmt.Println("Preparing status packet")
  status := structs.CreateNewStatusPacket(gossiper.Want)
  statusPacket := structs.GossipPacket{Status: status}
  packetAndAddress := structs.PacketAndAddress{Packet: &statusPacket, SenderAddr: gossiper.Address.String()}
  // send the rumor message to the randomly chosen peer through the corresponding chanel
  gossiper.MapOfChanels[senderAddr] <- packetAndAddress
}

func handleIncomingStatusPacket(gossiper *structs.Gossiper, packet *structs.GossipPacket, senderAddr string) {
  fmt.Println("RECEIVED MY FIRST STATUS PACKET!!!")
}
