package gossiper

import "net"
import "fmt"
import "math/rand"
import "time"
import "strings"
import "github.com/JohnDoe/Peerster/structs"
import "github.com/JohnDoe/Peerster/helpers"
import "github.com/dedis/protobuf"


// Function to simply broadcast a newly received message (client or peer) to all known peers
func broadcastGossipPacket (gossiper *structs.Gossiper, gossipPacket *structs.GossipPacket, peerSenderAddress string) {
    knownPeers := helpers.JoinMapKeys(gossiper.Peers)

    if (len(knownPeers) != 0) {
      listOfPeers := strings.Split(knownPeers, ",")

      for _, peer := range listOfPeers {
        if peer != peerSenderAddress {
          sendPacket(gossiper, gossipPacket, peer)
      }
    }
  }
}

func sendPacket(gossiper *structs.Gossiper, gossipPacket *structs.GossipPacket, addressOfReceiver string) {

    udpAddr, err := net.ResolveUDPAddr("udp4", addressOfReceiver)
    if err != nil {
      fmt.Println("Error resolving udp addres: ", err)
    }

    packetBytes, err := protobuf.Encode(gossipPacket)
    if err != nil {
      fmt.Println("Error encoding a simple message: ", err)
    }

    gossiper.Conn.WriteToUDP(packetBytes, udpAddr)
}

func chooseRandomPeerAndSendPacket (gossiper *structs.Gossiper, gossipPacket *structs.GossipPacket, peerSenderAddress string) {

  if gossipPacket.Rumor == nil {
    fmt.Println("Trying to send a null RumorMessage")
    return
  }

  knownPeers := helpers.JoinMapKeys(gossiper.Peers)
  if (len(knownPeers) != 0) {
    listOfPeers := strings.Split(knownPeers, ",")
    // pick a random peer to send the message to
    seed := rand.NewSource(time.Now().UnixNano())
    rng := rand.New(seed)
    idx := rng.Intn(len(listOfPeers))
    chosenPeer := listOfPeers[idx]
    sendPacket(gossiper, gossipPacket, chosenPeer)
  }
}


func sendAcknowledgementStatusPacket (gossiper *structs.Gossiper, peerSenderAddress string) {
  statusPacket := structs.StatusPacket{Want: gossiper.Want}
  gossipPacket := structs.GossipPacket{Status: &statusPacket}
  sendPacket(gossiper, &gossipPacket, peerSenderAddress)
}
