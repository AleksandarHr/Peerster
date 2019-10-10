package gossiper

import "net"
import "fmt"
import "math/rand"
import "time"
import "strings"
import "github.com/JohnDoe/Peerster/structs"
import "github.com/JohnDoe/Peerster/helpers"
import "github.com/dedis/protobuf"


func chooseRandomPeerAndSendPacket (gossiper *structs.Gossiper, gossipPacket *structs.GossipPacket, peerSenderAddress string) string {

  chosenPeer := ""
  if gossipPacket.Rumor == nil {
    fmt.Println("Trying to send a null RumorMessage")
    return chosenPeer
  }

  knownPeers := helpers.JoinMapKeys(gossiper.Peers)
  if (len(knownPeers) != 0) {
    listOfPeers := strings.Split(knownPeers, ",")
    // pick a random peer to send the message to
    seed := rand.NewSource(time.Now().UnixNano())
    rng := rand.New(seed)
    idx := rng.Intn(len(listOfPeers))
    chosenPeer = listOfPeers[idx]
    // sendPacket(gossiper, gossipPacket, chosenPeer)
  }
  return chosenPeer
}

func sendAcknowledgementStatusPacket (gossiper *structs.Gossiper, peerSenderAddress string) {
  statusPacket := structs.StatusPacket{Want: gossiper.Want}
  gossipPacket := structs.GossipPacket{Status: &statusPacket}
  sendPacket(gossiper, &gossipPacket, peerSenderAddress)
}

// When the original sender receives a status packet from the receiver, compare its own status to it
//    and return ' the first ' rumor which the receiver does not have
func getNextRumorToSendIfSenderHasOne(gossiper *structs.Gossiper, receivedStatus *structs.StatusPacket) *structs.RumorMessage {

  gossiperStatusMap := helpers.ConvertPeerStatusVectorClockToMap(gossiper.Want)
  receivedStatusMap := helpers.ConvertPeerStatusVectorClockToMap(receivedStatus.Want)
  var returnStatus *structs.PeerStatus = nil
  // if the
  for k, v := range gossiperStatusMap {
    if val, ok := receivedStatusMap[k]; !ok {
      // the gossiper/sender node has at least one message with an origin k while
      //    the receiver node has none at all - send rumor (k, 1)
      returnStatus.Identifier = k
      returnStatus.NextID = 1
    } else {
      // both gossiper/sender node and the receiver node have at least one message
      //    with an origin k. compare the NextID
      if v > val {
        // gossiper/sender has a newer message with origin k than the receiver node does
        returnStatus.Identifier = k
        returnStatus.NextID = v
      }
    }
  }

  return getRumorFromSeenMessages(gossiper, returnStatus)
}

func getRumorFromSeenMessages(gossiper *structs.Gossiper, status *structs.PeerStatus) *structs.RumorMessage {
  gossiper.MyMessages.Lck.Lock()
  defer gossiper.MyMessages.Lck.Unlock()
  var returnRumor *structs.RumorMessage = nil

  if messagesFromOrigin, ok := gossiper.MyMessages.Messages[status.Identifier]; ok {
    for _, msg := range messagesFromOrigin {
      if msg.ID == status.NextID {
        returnRumor = &msg
      }
    }
  }
  return returnRumor
}
//============================================================================================================
//============================================================================================================
//                                SIMPLE MODE FUNCTIONS AND HELPERS
//============================================================================================================
//============================================================================================================
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
