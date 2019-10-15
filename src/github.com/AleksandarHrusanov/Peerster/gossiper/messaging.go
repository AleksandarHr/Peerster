package gossiper

import "net"
import "fmt"
import "math/rand"
import "time"
import "strings"
import "github.com/AleksandarHrusanov/Peerster/structs"
import "github.com/AleksandarHrusanov/Peerster/helpers"
import "github.com/dedis/protobuf"


func chooseRandomPeer (gossiper *structs.Gossiper) string {

  chosenPeer := ""
  knownPeers := helpers.JoinMapKeys(gossiper.Peers)
  if (len(knownPeers) != 0) {
    listOfPeers := strings.Split(knownPeers, ",")
    // pick a random peer to send the message to
    seed := rand.NewSource(time.Now().UnixNano())
    rng := rand.New(seed)
    idx := rng.Intn(len(listOfPeers))
    chosenPeer = listOfPeers[idx]
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
func getStatusForNextRumor(gossiperStatus *[]structs.PeerStatus, receivedStatus *[]structs.PeerStatus) *structs.PeerStatus {

  gossiperStatusMap := helpers.ConvertPeerStatusVectorClockToMap(*gossiperStatus)
  receivedStatusMap := helpers.ConvertPeerStatusVectorClockToMap(*receivedStatus)
  returnStatus := structs.PeerStatus{Identifier: "", NextID: 0}
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
        returnStatus.Identifier = k
        returnStatus.NextID = val
      }
    }
  }
  return &returnStatus
}

func getRumorFromSeenMessages(gossiper *structs.Gossiper, status *structs.PeerStatus) *structs.RumorMessage {

  if status.Identifier == "" || status.NextID == 0 {
    return nil;
  }

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
