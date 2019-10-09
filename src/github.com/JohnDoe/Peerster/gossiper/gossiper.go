package gossiper

import "net"
import "fmt"
import "time"
import "github.com/dedis/protobuf"
import "github.com/JohnDoe/Peerster/helpers"
import "github.com/JohnDoe/Peerster/structs"


var maxBufferSize = 1024
var localhost = "127.0.0.1"

/*HandleChanelMap - a function to keep the [addres -> chanel] map up to date
*/
func HandleChanelMap(gossiper *structs.Gossiper, mapCh chan string) {
  for {
    select {
    case addressToAdd := <- mapCh:
      if _, ok := gossiper.MapOfChanels[addressToAdd]; !ok {
        newChanel := make(chan structs.PacketAndAddress)
        gossiper.MapOfChanels[addressToAdd] = newChanel
      }
    }
  }
}

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
    gossipPacket := &structs.GossipPacket{}

    if simpleFlag {
      // If the simple flag IS on, create a SimpleMessage from the user message
      simpleMessage := structs.CreateNewSimpleMessage(gossiper.Name, gossiper.Address.String(), clientMessage)
      gossipPacket.Simple = simpleMessage
      // Broadcast the client message to all known peers
      broadcastGossipPacket(gossiper, gossipPacket, "")
    } else {
      // If simple flag IS NOT on, create a RumorMessage from the user message
      rumorMessage := structs.CreateNewRumorMessage(gossiper.Name, uint32(1), clientMessage)
      gossipPacket.Rumor = rumorMessage

      // DO I HAVE TO CHECK FOR SPAM (this gossiper receiving the same TEXT from a client)?
      peerStatus := structs.CreateNewPeerStatusPair(gossiper.Name, uint32(rumorMessage.ID + 1))
      updatePeerStatusList(gossiper, peerStatus)
      // ADD new rumor message to SEEN MESSAGES
      updateSeenMessages(gossiper, rumorMessage)

      // BEGIN RUMORMONGERING in a go routine
      go initiateRumorMongering(gossiper, gossipPacket)
    }
  }
}


func initiateRumorMongering(gossiper *structs.Gossiper, packet *structs.GossipPacket) {
  // Choose random peer to send the rumor message to and add to the map of chanels
  chosenPeer := chooseRandomPeerAndSendPacket (gossiper, packet , gossiper.Address.String())
  fmt.Println("Initiating rumor mongering with peer: ", chosenPeer)
  gossiper.MapHandler <- chosenPeer

  // ??? how to make sure the map of chanels has been updated and the new chanel has been created on time ???
  rumor := packet.Rumor
  addr := gossiper.Address.String()
  fmt.Println("Rumor text = ", rumor.Text)
  if rumor != nil {
    packetAndAddress := structs.PacketAndAddress{Packet: packet, SenderAddr: addr}
    // send the rumor message to the randomly chosen peer through the corresponding chanel
    fmt.Println("SENDING A PACKET TO THE CHANEL")
    gossiper.MapOfChanels[chosenPeer] <- packetAndAddress
  }
}



/*HandleGossipPackets - a function to handle incoming gossip packets - simple packet, rumor packet, status packet */
func HandleGossipPackets(gossiper *structs.Gossiper, simpleFlag bool, incomingPacketsChannel chan structs.PacketAndAddress) {

  // 1) open a go routine to with a channel aceepting packets, which will call the appropriate handling function
  // a go routine which loops forever, accepts incoming packets, and distributes them
  //    to helper functions depending on their type (simple, rumor, status)
  go func(msgChannel chan structs.PacketAndAddress) {
    for{
      select {
      case receivedPacketAndSenderAddr := <- msgChannel:
        receivedPacket := receivedPacketAndSenderAddr.Packet
        senderAddr := receivedPacketAndSenderAddr.SenderAddr
        fmt.Println("Gossiper ", gossiper.Address.String(), " received a message from node ", senderAddr)

        // if the simple flag is on, only handle simple packets and disregard any others
        if simpleFlag {
          if (receivedPacket.Simple != nil) {
            handleIncomingSimplePacket(gossiper, receivedPacket, senderAddr)
          }
        } else if receivedPacket.Rumor != nil {
          fmt.Println("It is a rumor packet")
          handleIncomingRumorPacket(gossiper, receivedPacket, senderAddr)
        } else if receivedPacket.Status != nil {
          handleIncomingStatusPacket(gossiper, receivedPacket, senderAddr)
        }
      }
    }
  }(incomingPacketsChannel)

  // 2) open a go routine for the infinite loop of accepting incoming messages
  // a go routine which loops forever, reads incoming messages and sends them to
  // the previous go routine to be handled accordingly
  go func(msgChanel chan structs.PacketAndAddress) {
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

      // 3) whenever a new message arrives in routine2, send it via channel to routine1 which will handle it
      senderAddr := addr.String()
      msgChanel <- structs.PacketAndAddress{Packet: &packet, SenderAddr: senderAddr}
    }
  }(incomingPacketsChannel)
}



// =======================================================================================
// =======================================================================================
//                                Helper Functions
// =======================================================================================
// =======================================================================================

func updateSeenMessages (gossiper *structs.Gossiper, newRumor *structs.RumorMessage) {

  gossiper.MyMessages.Lck.Lock()
  defer gossiper.MyMessages.Lck.Unlock()

  origin := newRumor.Origin
  currentMessages := gossiper.MyMessages.Messages[origin]
  currentMessages = append(currentMessages, *newRumor)
  gossiper.MyMessages.Messages[origin] = currentMessages
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

// Code written based on the example in 'golang.org/pkg/time'
func mongeringTimeout () {
  ticker := time.NewTicker(time.Second)
  defer ticker.Stop()
  // SUBSTITUTE THIS WITH A CHANNEL WHERE THE GOSSIPER RECEIVES MESSAGES
  gossiperChannel := make(chan bool)
  go func() {
    time.Sleep(10 * time.Second)
    gossiperChannel <- true
  }()
  // for {
  //   select{
  //   case status := <- gossiperChannel:
  //     // if a status packet is received through the gossiper's channel
  //     // do as neededs
  //     fmt.Println("HANDLE STATUS PACKET")
  //     // Restart timer
  //     ticker = time.NewTicker(time.Second)
  //   case t := <-ticker.C:
  //     fmt.Println("Current rumor mongering timed out.")
  //     fmt.Println("Pick another peer to start rumor mongering with.")
  //     // RANDOMLY PICK A
  //   }
  // }
}
