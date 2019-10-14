package gossiper

import "net"
import "fmt"
import "time"
import "github.com/dedis/protobuf"
import "github.com/JohnDoe/Peerster/helpers"
import "github.com/JohnDoe/Peerster/structs"

var maxBufferSize = 1024
var localhost = "127.0.0.1"
var sciper = "309750"

/*HandleChanelMap - a function to keep the [addres -> chanel] map up to date
*/
func HandleChanelMap(gossiper *structs.Gossiper, mapCh chan string) {
  for {
    select {
    case addressOfCommunication := <- mapCh:
      if _, ok := gossiper.MapOfChanels[addressOfCommunication]; !ok {
        newChanel := make(chan structs.PacketAndAddresses, 1)
        gossiper.MapOfChanels[addressOfCommunication] = newChanel
        go handleCommunicationOnNewChanel(gossiper, addressOfCommunication)
      }
    }
  }
}

func handleCommunicationOnNewChanel (gossiper *structs.Gossiper, newAddress string) {
  newChanel := gossiper.MapOfChanels[newAddress]
  for {
    select{
    case packet := <- newChanel:
      if packet.SenderAddr == gossiper.Address.String() {
        // current node is SENDING a packet to addressToAdd
        sendPacket(gossiper, packet.Packet, newAddress)
      } else if packet.ReceiverAddr == gossiper.Address.String() {
        // current node is RECEIVING a packet from addressToAdd
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
      rumorMessage := structs.CreateNewRumorMessage(gossiper.Name + sciper, gossiper.CurrentMessageID, clientMessage)
      gossiper.CurrentMessageID++;
      gossipPacket.Rumor = rumorMessage

      peerStatus := structs.CreateNewPeerStatusPair(rumorMessage.Origin, uint32(rumorMessage.ID + 1))
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
  chosenPeer := chooseRandomPeer(gossiper)
  if chosenPeer == "" {
    fmt.Println("Current gossiper node has no known peers and cannot initiate rumor mongering.")
    return
  }

  gossiper.MapHandler <- chosenPeer
  time.Sleep(2*time.Second)

  go sendRumorAndWaitForStatusOrTimeout(gossiper, packet, chosenPeer)
}


func sendRumorAndWaitForStatusOrTimeout (gossiper *structs.Gossiper, packet *structs.GossipPacket, receiverAddr string) {
  rumor := packet.Rumor
  addr := gossiper.Address.String()
  if rumor != nil {
    gossiper.MongeringMessages[receiverAddr] = *rumor
    pckt := structs.PacketAndAddresses{Packet: packet, SenderAddr: addr, ReceiverAddr: receiverAddr}
    // send the rumor message to the randomly chosen peer through the corresponding chanel
    gossiper.MapOfChanels[receiverAddr] <- pckt
    fmt.Println("Just sent the packet to the chanel - write to stdout??")
    helpers.WriteToStandardOutputWhenMongering(receiverAddr)
    mongeringTimeout(gossiper, receiverAddr, packet)
  }
}

// Code written based on the example in 'golang.org/pkg/time'
func mongeringTimeout (gossiper *structs.Gossiper, chosenPeer string, packet *structs.GossipPacket) {
    for {// TIMEOUT
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()
    timeoutChanel := make(chan bool)
    go func() {
      time.Sleep(5 * time.Second)
      timeoutChanel <- true
    }()
    select{
    case status := <- gossiper.MapOfChanels[chosenPeer]:
      // if a status packet is received through the gossiper's channel
      // do as needed
      gossiper.PacketChanel <- status
      // Restart timer
      ticker = time.NewTicker(time.Second)
    case t := <- timeoutChanel:
      if t {
        fmt.Println("Timeout occured")
        go initiateRumorMongering(gossiper, packet)
        break;
      }
    }
  }
  fmt.Println("EXITTED MONGERING TIMEOUT LOOP")
  }

/*HandleGossipPackets - a function to handle incoming gossip packets - simple packet, rumor packet, status packet */
func HandleGossipPackets(gossiper *structs.Gossiper, simpleFlag bool, incomingPacketsChannel chan structs.PacketAndAddresses) {
  // 1) open a go routine for the infinite loop of accepting incoming messages
  // a go routine which loops forever, reads incoming messages and sends them to
  // the following go routine to be handled accordingly
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
      // 3) whenever a new message arrives in routine1, send it via channel to routine2 which will handle it
      senderAddr := addr.String()
      msgChanel <- structs.PacketAndAddresses{Packet: &packet, SenderAddr: senderAddr, ReceiverAddr: gossiper.Address.String()}
    }
  }(incomingPacketsChannel)


  // 2) open a go routine to with a channel aceepting packets, which will call the appropriate handling function
  // a go routine which loops forever, accepts incoming packets, and distributes them
  //    to helper functions depending on their type (simple, rumor, status)
  go func(msgChannel chan structs.PacketAndAddresses) {
    for{
      select {
      case receivedPacketAndAddresses := <- msgChannel:
        receivedPacket := receivedPacketAndAddresses.Packet
        senderAddr := receivedPacketAndAddresses.SenderAddr
        receiverAddr := receivedPacketAndAddresses.ReceiverAddr

        if receiverAddr == gossiper.Address.String() {
          // Current node is receiving a packet
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
        } else if senderAddr == gossiper.Address.String() {
          // Current node is sending out a packet
          sendPacket(gossiper, receivedPacket, receiverAddr)
        }
      }
    }
  }(incomingPacketsChannel)
}

/*HandleAntiEntropy - a function to fire periodically gossiper's status to a random known peer */
func HandleAntiEntropy(gossiper *structs.Gossiper) {
  for {
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()
    timeoutChanel := make(chan bool)
    go func() {
      time.Sleep(10 * time.Second)
      timeoutChanel <- true
    }()
    select{
    case t := <- timeoutChanel:
      if t {
        fmt.Println("Anti entropy timeout occured")
        sp := &structs.StatusPacket{Want: gossiper.Want}
        pckt := &structs.GossipPacket{Status: sp}
        chosenPeer := chooseRandomPeer(gossiper)
        if chosenPeer == "" {
          fmt.Println("Gossiper has no known peers, cannot execute anti entropy")
        } else {
          gossiper.MapHandler <- chosenPeer
          sendPacket(gossiper, pckt, chosenPeer)
        }
      }
    }
  }
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
      alreadyExisted = true
    }
  }
  // if an entry with the same origin does not exist, append the status object to the end of the slice
  if !alreadyExisted {
    gossiper.Want = append(gossiper.Want, *status)
  }
}
