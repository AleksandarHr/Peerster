package gossiper

import "flag"
import "net"
import "fmt"
import "strings"
import "time"
import "math/rand"
import "github.com/dedis/protobuf"
import "github.com/JohnDoe/Peerster/helpers"


var maxBufferSize = 1024
var localhost = "127.0.0.1"

/*NewGossiper - a function acting as a constructor for a the Gossiper struct, returns a pointer
  - address, string - the address for the gossiper node in the form 'ip:port'
  - flags, *FlagsInformation - a pointer to a FlagsInformation object
*/
func NewGossiper(address string, flags *helpers.FlagsInformation) *helpers.Gossiper {
  udpAddr, err := net.ResolveUDPAddr("udp4", address)
  if err != nil {
    fmt.Println("Error resolving udp addres: ", err)
  }

  udpConn, err := net.ListenUDP("udp4", udpAddr)
  if err != nil {
    fmt.Println("Error listening: ", err)
  }

  peers := make(map[string]bool)
  for _, p := range (strings.Split(flags.Peers, ",")){
    peers[p] = true
  }

  var knownMessages []helpers.PeerStatus

  fmt.Println("Gossiper is up and listening on ", address)

  return &helpers.Gossiper {
    Address:  udpAddr,
    Conn:     udpConn,
    Name:     flags.Name,
    Peers:    peers,
    Want:     knownMessages,
  }
}

/*HandleClientMessages - A function to handle messages coming from a client
    * gossiper *Gossiper - poitner to a gossiper
    * uiPort string - the uiPort of the current gossiper
*/
func HandleClientMessages(gossiper *helpers.Gossiper, uiPort string, simpleFlag bool) {

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
    gossipPacket := helpers.GossipPacket{}

    if simpleFlag {
      // If the simple flag IS on, create a SimpleMessage from the user message
      simpleMessage := helpers.SimpleMessage{}
      simpleMessage.OriginalName = gossiper.Name
      simpleMessage.RelayPeerAddr = gossiper.Address.String()
      simpleMessage.Contents = clientMessage
      gossipPacket.Simple = &simpleMessage
      // Broadcast the client message to all known peers
      broadcastGossipPacket(gossiper, gossipPacket, "")
    } else {
      // If simple flag IS NOT on, create a RumorMessage from the user message
      rumorMessage := helpers.RumorMessage{}
      rumorMessage.Origin = gossiper.Name
      rumorMessage.ID = 1
      rumorMessage.Text = clientMessage
      gossipPacket.Rumor = &rumorMessage

      peerStatus := helpers.PeerStatus{Identifier: gossiper.Name, NextID : uint32(rumorMessage.ID + 1)}
      updatePeerStatusList(gossiper, &peerStatus)
      // Rumormonger the client message in the form of a GossipPacket (e.g. SimpleMessage or RumorMessage)
      rumorMongering(gossiper, gossipPacket, "")
    }
  }
}

/*HandlePeerMessages - a function to handle messages coming from a client
    * gossiper *Gossiper - poitner to a gossiper
*/
func HandlePeerMessages (gossiper *helpers.Gossiper, simpleFlag bool) {
  // Goroutine (thread) to handle incoming messages from other gossipers
  peerBuffer := make([]byte, maxBufferSize)
  for {
    numBytes, addr, errRead := gossiper.Conn.ReadFromUDP(peerBuffer)
    if errRead != nil {
      fmt.Println("Error reading from a peer message from UDP: ", errRead)
    }
    packet := helpers.GossipPacket{}
    errDecode := protobuf.Decode(peerBuffer[:numBytes], &packet)
    if errDecode != nil {
      fmt.Println("Error decoding message: ", errDecode)
    }

    senderAddress := addr.String()
    if (packet.Simple != nil) && simpleFlag {
      // Handle a SimpleMessage from a peer
      senderAddress = packet.Simple.RelayPeerAddr
      // Store the RelayPeerAddr in the map of known peers
      gossiper.Peers[senderAddress] = true
      // Write received peer message to standard output
      helpers.WriteToStandardOutputWhenPeerSimpleMessageReceived(gossiper, &packet)
      // Change the RelayPeerAddr to the current gossiper node's address
      packet.Simple.RelayPeerAddr = gossiper.Address.String()
      broadcastGossipPacket(gossiper, packet, senderAddress)
    } else if packet.Rumor != nil {
      // Handle a RumorMessage from a peer
      helpers.WriteToStandardOutputWhenRumorMessageReceived(gossiper, &packet, senderAddress)
      sendStatusMessage(gossiper)
    } else if packet.Status != nil {
      // Handle a StatusMessage from a peer
      rumorMongering(gossiper, packet, senderAddress)
    }
  }
}

func rumorMongering (gossiper *helpers.Gossiper, gossipPacket helpers.GossipPacket, peerSenderAddress string) {
  knownPeers := helpers.JoinMapKeys(gossiper.Peers)
  if (len(knownPeers) != 0) {
    listOfPeers := strings.Split(knownPeers, ",")
    // pick a random peer to send the message to
    seed := rand.NewSource(time.Now().UnixNano())
    rng := rand.New(seed)
    idx := rng.Intn(len(listOfPeers))
    chosenPeer := listOfPeers[idx]
    fmt.Println("CHOSEN PEER ADDRES = ", chosenPeer)
    sendPacket(gossiper, gossipPacket, chosenPeer)
  }
}

func sendStatusMessage(gossiper *helpers.Gossiper) {

}

func sendPacket(gossiper *helpers.Gossiper, gossipPacket helpers.GossipPacket, addressOfReceiver string) {

    udpAddr, err := net.ResolveUDPAddr("udp4", addressOfReceiver)
    if err != nil {
      fmt.Println("Error resolving udp addres: ", err)
    }

    packetBytes, err := protobuf.Encode(&gossipPacket)
    if err != nil {
      fmt.Println("Error encoding a simple message: ", err)
    }

    gossiper.Conn.WriteToUDP(packetBytes, udpAddr)
}


// =======================================================================================
// =======================================================================================
//                                Helper Functions
// =======================================================================================
// =======================================================================================

/*HandleFlags - A function to handle flags passed to the gossiper as described at the beginning of the file
*/
func HandleFlags() (*helpers.FlagsInformation) {
  // Read all the flags
  var UIPortFlag = flag.String("UIPort", "8080", "port for the UI client")
  var gossipAddrFlag = flag.String("gossipAddr", localhost + ":" + "5000", "ip:port for the gossiper")
  var nameFlag = flag.String("name", "new_node", "name of the gossiper")
  var peersFlag = flag.String("peers", "", "comma separated list of peers of the form ip:port")
  var simpleFlag = flag.Bool("simple", true, "run gossiper in simple mode")

  // Parse all flagse
  flag.Parse()

  // Save the flags information
  port := *UIPortFlag;
  gossipAddr := *gossipAddrFlag;
  name := *nameFlag;
  peers := *peersFlag;
  simple := *simpleFlag;

  flagsInfo := helpers.FlagsInformation{UIPort : port, GossipAddress : gossipAddr, Name : name, Peers : peers, Simple : simple}
  return &flagsInfo
}


func updatePeerStatusList(gossiper *helpers.Gossiper, status *helpers.PeerStatus) {
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


// Function to simply broadcast a newly received message (client or peer) to all known peers
func broadcastGossipPacket (gossiper *helpers.Gossiper, gossipPacket helpers.GossipPacket, peerSenderAddress string) {
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
