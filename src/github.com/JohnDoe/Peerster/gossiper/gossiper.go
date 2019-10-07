package gossiper

import "net"
import "fmt"
import "github.com/dedis/protobuf"
import "github.com/JohnDoe/Peerster/helpers"
import "github.com/JohnDoe/Peerster/structs"


var maxBufferSize = 1024
var localhost = "127.0.0.1"

/*HandleClientMessages - A function to handle messages coming from a client
    * gossiper *Gossiper - poitner to a gossiper
    * uiPort string - the uiPort of the current gossiper
*/
func HandleClientMessages(gossiper *structs.Gossiper, uiPort string, simpleFlag bool) {

  // Resolve uiAddress and listen for incoming UDP messages from clients
  // NOTE: We assume that the client runs locally, so client's address is "127.0.0.1:UIPort"
  uiAddress := localhost + ":" + uiPort
  uiAddr, errResolve := net.ResolveUDPAddr("udp4", uiAddress)
  if errResolve != nil {
    fmt.Println("Error resolving udp addres: ", errResolve)
  }
  uiConn, errListen := net.ListenUDP("udp4", uiAddr)
  if errListen != nil {
    fmt.Println("Error listening: ", errListen)
  }

  // Create a buffer for client messages and start an infinite for loop reading incoming messages
  clientBuffer := make([]byte, maxBufferSize)
  for {
    // Read incoming client message
    numBytes, conn, errRead := uiConn.ReadFromUDP(clientBuffer)
    if errRead != nil {
      fmt.Println("Error reading from UDP: ", errRead)
    }
    if conn == nil {
      continue
    }

    // Write received client message to standard output
    clientMessage := string(clientBuffer[0:numBytes])
    helpers.WriteToStandardOutputWhenClientMessageReceived(gossiper, clientMessage)
    gossipPacket := structs.GossipPacket{}

    if simpleFlag {
      // If the simple flag IS on, create a SimpleMessage from the user message
      simpleMessage := structs.SimpleMessage{}
      simpleMessage.OriginalName = gossiper.Name
      simpleMessage.RelayPeerAddr = gossiper.Address.String()
      simpleMessage.Contents = clientMessage
      gossipPacket.Simple = &simpleMessage
      // Broadcast the client message to all known peers
      broadcastGossipPacket(gossiper, &gossipPacket, "")
    } else {
      // If simple flag IS NOT on, create a RumorMessage from the user message
      rumorMessage := structs.RumorMessage{}
      rumorMessage.Origin = gossiper.Name
      rumorMessage.ID = 1
      rumorMessage.Text = clientMessage
      gossipPacket.Rumor = &rumorMessage

      // DO I HAVE TO CHECK FOR SPAM (this gossiper receiving the same TEXT from a client)?
      peerStatus := structs.PeerStatus{Identifier: gossiper.Name, NextID : uint32(rumorMessage.ID + 1)}
      updatePeerStatusList(gossiper, &peerStatus)
      // Rumormonger the client message in the form of a GossipPacket (e.g. SimpleMessage or RumorMessage)
      chooseRandomPeerAndSendPacket(gossiper, &gossipPacket, "")
    }
  }
}

/*HandlePeerMessages - a function to handle messages coming from a client
    * gossiper *Gossiper - poitner to a gossiper
*/
func HandlePeerMessages (gossiper *structs.Gossiper, simpleFlag bool) {
  // Goroutine (thread) to handle incoming messages from other gossipers
  peerBuffer := make([]byte, maxBufferSize)
  for {
    numBytes, addr, errRead := gossiper.Conn.ReadFromUDP(peerBuffer)
    if errRead != nil {
      fmt.Println("Error reading from a peer message from UDP: ", errRead)
    }
    packet := structs.GossipPacket{}
    errDecode := protobuf.Decode(peerBuffer[:numBytes], &packet)
    if errDecode != nil {
      fmt.Println("Error decoding message: ", errDecode)
    }

    senderAddress := addr.String()
    //currentRummor := structs.RumorMessage{}

    if simpleFlag {
      if packet.Simple != nil {
        // Handle a SimpleMessage from a peer
        senderAddress = packet.Simple.RelayPeerAddr
        // Store the RelayPeerAddr in the map of known peers
        gossiper.Peers[senderAddress] = true
        // Write received peer message to standard output
        helpers.WriteToStandardOutputWhenPeerSimpleMessageReceived(gossiper, &packet)
        // Change the RelayPeerAddr to the current gossiper node's address
        packet.Simple.RelayPeerAddr = gossiper.Address.String()
        broadcastGossipPacket(gossiper, &packet, senderAddress)
      }
    } else {
      if packet.Rumor != nil {
        // Handle a RumorMessage from a peer
        rumor := packet.Rumor
        // if message has been previously received, disregard it
        messageSeen := helpers.AlreadySeenMessage(gossiper, rumor)
        if !messageSeen {
          //currentRummor = *rumor
          // if message is new, write to standard output
          helpers.WriteToStandardOutputWhenRumorMessageReceived(gossiper, &packet, senderAddress)

          // update current gossiper's PeerStatus structure
          peerStatus := structs.PeerStatus{Identifier: rumor.Origin, NextID : uint32(rumor.ID + 1)}
          updatePeerStatusList(gossiper, &peerStatus)

          // send a status packet back to the sender to acknowledge receiving the rumor message
          sendAcknowledgementStatusPacket(gossiper, senderAddress)

          // choose a random known peer and send them the packet
          chooseRandomPeerAndSendPacket(gossiper, &packet, senderAddress)
          // IMPLEMENT TIMEOUT FOR THE STATUS PACKET
        }
      } else if packet.Status != nil {
        receivedStatus := packet.Status
        // Handle a StatusMessage from a peer
        // compare gossiper's PeerStatus with the received StatusPacket
        potentialNextMessageToSend := getNextRumorToSendIfSenderHasOne(gossiper, receivedStatus)
        if potentialNextMessageToSend != nil {
          // sender peer S has other messages that the receiver peer R does not, send the 'first such message'
        } else {
          // sender peer S has no other messages that the receiver peer R does not, but the receiver peer R
          //    has messages that the sender peer S does not. sender peer S sends a StatusPacket to receiver peer R
          sendAcknowledgementStatusPacket(gossiper, senderAddress)
        }
        chooseRandomPeerAndSendPacket(gossiper, &packet, senderAddress)
      }
    }
  }
}

func getNextRumorToSendIfSenderHasOne(gossiper *structs.Gossiper, receivedStatus *structs.StatusPacket) *structs.PeerStatus {

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

  return returnStatus
}


// =======================================================================================
// =======================================================================================
//                                Helper Functions
// =======================================================================================
// =======================================================================================



func updatePeerStatusList(gossiper *structs.Gossiper, status *structs.PeerStatus) {
  alreadyExisted := false
  // Iterate over the vector lock - if an entry with the same origin exists, substitute with the new status
  for i := 0; i < len(gossiper.Want); i++ {
    if gossiper.Want[i].Identifier == status.Identifier {
      gossiper.Want[i] = *status
    }
  }
  // if an entry with the same origin does not exist, append the status object to the end of the slice
  if !alreadyExisted {
    gossiper.Want = append(gossiper.Want, *status)
  }
}
