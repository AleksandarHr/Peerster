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

  if helpers.AlreadySeenMessage(gossiper, rumor) {
    // received an already seen message
  } else {
    // If message is new, print to standard output, update seen messages and chanel map,
    helpers.WriteToStandardOutputWhenRumorMessageReceived(gossiper, packet, senderAddr)
    // If this is the first rumor packet of a rumormongering session, update the map of chanels of this node
    gossiper.MapHandler <- senderAddr
    updateSeenMessages(gossiper, rumor)
    // Update PeerStatus information
    peerStatus := structs.CreateNewPeerStatusPair(rumor.Origin, uint32(rumor.ID + 1))
    updatePeerStatusList(gossiper, peerStatus)

    // Send status packet
    status := structs.CreateNewStatusPacket(gossiper.Want)
    statusPacket := structs.GossipPacket{Status: status}
    // PacketAndAddresses := structs.PacketAndAddresses{Packet: &statusPacket, SenderAddr: gossiper.Address.String()}
    // send the rumor message to the randomly chosen peer through the corresponding chanel
    sendPacket(gossiper, &statusPacket, senderAddr)
  }
}

func handleIncomingStatusPacket(gossiper *structs.Gossiper, packet *structs.GossipPacket, senderAddr string) {
  status := packet.Status
  helpers.WriteToStandardOutputWhenStatusMessageReceived(gossiper, packet, senderAddr)

  newRumorStatuts := getStatusForNextRumor(&gossiper.Want, &status.Want)
  newRumorToSend := getRumorFromSeenMessages(gossiper, newRumorStatuts)
  if newRumorToSend != nil {
    // Sender has more rumors to send
    fmt.Println("I have something to send!! ")
    nextPacket := structs.GossipPacket{Rumor: newRumorToSend}
    go sendRumorAndWaitForStatusOrTimeout(gossiper, &nextPacket, senderAddr)
  } else {
    fmt.Println("I need something you have - send it to me!!")
    // check if the original sender has seen all of the original receiver's messages
    reverseSendingRumorStatus := getStatusForNextRumor(&status.Want, &gossiper.Want)
    if reverseSendingRumorStatus.Identifier != "" && reverseSendingRumorStatus.NextID != 0 {
      // original sender has not seen all of receiver's messages so sends a status packet to it
      statusToSendBack := structs.CreateNewStatusPacket(gossiper.Want)
      statusPacket := structs.GossipPacket{Status: statusToSendBack}
      sendPacket(gossiper, &statusPacket, senderAddr)
    } else {
      helpers.WriteToStandardOutputWhenInSyncWith(senderAddr)
      // statuses of both sender and receiver are the same - flip a coin
      coinResult := helpers.FlipCoin()
      if coinResult == 0 {
        //Pick a new peer to send the SAME rumor message to
        chosenPeer := chooseRandomPeer(gossiper)
        if chosenPeer == "" {
          fmt.Println("Current gossiper node has no known peers and cannot initiate rumor mongering.")
          return
        }
        helpers.WriteToStandardOutputWhenFlippedCoin(chosenPeer)
        // get the last rumor message that was sent to the peer we just found out are in sync
        lastRumor := gossiper.MongeringMessages[senderAddr]
        pckt := structs.GossipPacket{Rumor: &lastRumor}
        go initiateRumorMongering(gossiper, &pckt)
      } else if coinResult == 1 {
        fmt.Println("Exiting mongering process after flipping a coin")
        //End of rumor mongering process
      }
    }
  }
}
