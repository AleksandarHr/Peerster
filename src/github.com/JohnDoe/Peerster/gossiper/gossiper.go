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
        newChanel := make(chan structs.PacketAndAddresses)
        gossiper.MapOfChanels[addressToAdd] = newChanel
      }
      go handleCommunicationOnNewChanel(gossiper, addressToAdd)
    }
  }
}

func handleCommunicationOnNewChanel (gossiper *structs.Gossiper, newAddress string) {
  newChanel := gossiper.MapOfChanels[newAddress]
  for {
    select{
    case packet := <- newChanel:
      fmt.Println("Received a packet on the chanel")
      // current node is RECEIVING a packet from addressToAdd
      if packet.SenderAddr == gossiper.Address.String() {
        fmt.Println("SENDING")
        sendPacket(gossiper, packet.Packet, newAddress)
      } else {
        // current node is SENDING a packet to addressToAdd
        fmt.Println("RECEIVING")
        gossiper.PacketChanel <- packet
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
      rumorMessage := structs.CreateNewRumorMessage(gossiper.Name, gossiper.CurrentMessageID, clientMessage)
      gossiper.CurrentMessageID++;
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
  chosenPeer := chooseRandomPeerAndSendPacket(gossiper, packet , gossiper.Address.String())
  if chosenPeer == "" {
    fmt.Println("Current gossiper node has no known peers and cannot initiate rumor mongering.")
    return
  }

  gossiper.MapHandler <- chosenPeer
  fmt.Println("Initiating rumor mongering with peer: ", chosenPeer)
  time.Sleep(2*time.Second)

  go sendPacketAndWaitForStatus(gossiper, packet, chosenPeer)
}


func sendPacketAndWaitForStatus (gossiper *structs.Gossiper, packet *structs.GossipPacket, receiverAddr string) {
  rumor := packet.Rumor
  addr := gossiper.Address.String()
  if rumor != nil {
    PacketAndAddresses := structs.PacketAndAddresses{Packet: packet, SenderAddr: addr, ReceiverAddr: receiverAddr}
    // send the rumor message to the randomly chosen peer through the corresponding chanel
    gossiper.MapOfChanels[receiverAddr] <- PacketAndAddresses
    mongeringTimeout(gossiper, receiverAddr,packet)
  }
}

// Code written based on the example in 'golang.org/pkg/time'
func mongeringTimeout (gossiper *structs.Gossiper, chosenPeer string, packet *structs.GossipPacket) {
    fmt.Println("TIMEOUT CODE FOLLOWS")
    // TIMEOUT
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()
    // SUBSTITUTE THIS WITH A CHANNEL WHERE THE GOSSIPER RECEIVES MESSAGES
    timeoutChanel := make(chan bool)
    go func() {
      time.Sleep(10 * time.Second)
      timeoutChanel <- true
    }()
    for {
      fmt.Println("Waiting for a message from peer ", chosenPeer)
      fmt.Println("CURRENT GOSSIPER HAS SEEN ", len(gossiper.MyMessages.Messages[gossiper.Name]), " MESSAGES")
      select{
      case status := <- gossiper.MapOfChanels[chosenPeer]:
        // if a status packet is received through the gossiper's channel
        // do as needed
        fmt.Println("HANDLE STATUS PACKET from node ", status.SenderAddr)
        // Restart timer
        ticker = time.NewTicker(time.Second)
      case t := <- timeoutChanel:
        fmt.Println("Current rumor mongering timed out.", t)
        fmt.Println("Pick another peer to start rumor mongering with.")
        // RANDOMLY PICK A
        go initiateRumorMongering(gossiper, packet)
      }
    }
}

/*HandleGossipPackets - a function to handle incoming gossip packets - simple packet, rumor packet, status packet */
func HandleGossipPackets(gossiper *structs.Gossiper, simpleFlag bool, incomingPacketsChannel chan structs.PacketAndAddresses) {

  // 1) open a go routine to with a channel aceepting packets, which will call the appropriate handling function
  // a go routine which loops forever, accepts incoming packets, and distributes them
  //    to helper functions depending on their type (simple, rumor, status)
  go func(msgChannel chan structs.PacketAndAddresses) {
    for{
      select {
      case receivedPacketAndSenderAddr := <- msgChannel:
        receivedPacket := receivedPacketAndSenderAddr.Packet
        senderAddr := receivedPacketAndSenderAddr.SenderAddr
        fmt.Println("======== Packet arrived in HandleGossipPackets")
        fmt.Println("======== Address is = ", senderAddr)
          // Add to KnownPeers
          gossiper.Peers[senderAddr] = true
          // if the simple flag is on, only handle simple packets and disregard any others
          if simpleFlag {
            if (receivedPacket.Simple != nil) {
              handleIncomingSimplePacket(gossiper, receivedPacket, senderAddr)
            }
          } else if receivedPacket.Rumor != nil {
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
  go func(msgChanel chan structs.PacketAndAddresses) {
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
      msgChanel <- structs.PacketAndAddresses{Packet: &packet, SenderAddr: senderAddr, ReceiverAddr: gossiper.Address.String()}
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


func updatePeerStatusList(gossiper *structs.Gossiper, status *structs.PeerStatus) {
  alreadyExisted := false
  // Iterate over the vector clock - if an entry with the same origin exists, substitute with the new status
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
