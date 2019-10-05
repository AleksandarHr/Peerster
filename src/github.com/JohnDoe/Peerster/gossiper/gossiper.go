package gossiper

import "flag"
import "net"
import "fmt"
import "strings"
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

  fmt.Println("Gossiper is up and listening on ", address)

  return &helpers.Gossiper {
    Address:  udpAddr,
    Conn:     udpConn,
    Name:     flags.Name,
    Peers:    peers,
  }
}

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

/*HandlePeerMessages - a function to handle messages coming from a client
    * gossiper *Gossiper - poitner to a gossiper
*/
func HandlePeerMessages (gossiper *helpers.Gossiper) {
  // Goroutine (thread) to handle incoming messages from other gossipers
  peerBuffer := make([]byte, maxBufferSize)
  for {
    numBytes, _, _ := gossiper.Conn.ReadFromUDP(peerBuffer)
    packet := helpers.GossipPacket{}
    err := protobuf.Decode(peerBuffer[:numBytes], &packet)
    if err != nil {
      fmt.Println("Error decoding message: ", err)
    }

    senderAddress := packet.Simple.RelayPeerAddr
    // Store the RelayPeerAddr in the map of known peers
    gossiper.Peers[senderAddress] = true
    // Write received peer message to standard output
    writePeerMessageToStandardOutput(gossiper, packet)

    // Change the RelayPeerAddr to the current gossiper node's address
    packet.Simple.RelayPeerAddr = gossiper.Address.String()

    // Propagate the peer message in the form of a SimpleMessage to known peers
    propagateGossipPacket(gossiper, packet, senderAddress)
  }
}


/*HandleClientMessages - A function to handle messages coming from a client
    * gossiper *Gossiper - poitner to a gossiper
    * uiPort string - the uiPort of the current gossiper
*/
func HandleClientMessages(gossiper *helpers.Gossiper, uiPort string) {

  // Resolve uiAddress and listen for incoming UDP messages from clients
  // NOTE: We assume that the client runs locally, so client's address is "127.0.0.1:UIPort"
  uiAddress := localhost + ":" + uiPort
  uiAddr, err := net.ResolveUDPAddr("udp4", uiAddress)
  if err != nil {
    fmt.Println("Error resolving udp addres: ", err)
  }
  uiConn, err := net.ListenUDP("udp4", uiAddr)
  if err != nil {
    fmt.Println("Error listening: ", err)
  }

  // Create a buffer for client messages and start an infinite for loop reading incoming messages
  clientBuffer := make([]byte, maxBufferSize)
  for {
    // Read incoming client message
    numBytes, conn, err := uiConn.ReadFromUDP(clientBuffer)
    if err != nil {
      fmt.Println("could not read from UDP")
    }
    if conn == nil {
      continue
    }

    // Write received client message to standard output
    writeClientMessageToStandardOutput(gossiper, string(clientBuffer[0:numBytes]))

    // Create a simple message object
    simpleMessage := helpers.SimpleMessage{}
    simpleMessage.OriginalName = gossiper.Name
    simpleMessage.RelayPeerAddr = gossiper.Address.String()
    simpleMessage.Contents = string(clientBuffer[0:numBytes])
    gossipPacket := helpers.GossipPacket{Simple: &simpleMessage}

    // Propagate the client message in the form of a SimpleMessage to known peers
    propagateGossipPacket(gossiper, gossipPacket, "")
  }
}

func writeClientMessageToStandardOutput (gossiper *helpers.Gossiper, msg string) {
  fmt.Println("CLIENT MESSAGE " + msg)
  fmt.Println("PEERS " + helpers.JoinMapKeys(gossiper.Peers))
}

func writePeerMessageToStandardOutput (gossiper *helpers.Gossiper, packet helpers.GossipPacket) {
  fmt.Println("SIMPLE MESSAGE origin " +
              packet.Simple.OriginalName +
              " from " +
              packet.Simple.RelayPeerAddr +
              " contents " + packet.Simple.Contents)
  fmt.Println("PEERS " + helpers.JoinMapKeys(gossiper.Peers))
}

func propagateGossipPacket (gossiper *helpers.Gossiper, gossipPacket helpers.GossipPacket, peerSenderAddress string) {

  peers := helpers.JoinMapKeys(gossiper.Peers)

  if (len(peers) != 0) {
    listOfPeers := strings.Split(peers, ",")

    for _, peer := range listOfPeers {
      if peer != peerSenderAddress {
        udpAddr, err := net.ResolveUDPAddr("udp4", peer)
        if err != nil {
          fmt.Println("Error resolving udp addres: ", err)
        }

        packetBytes, err := protobuf.Encode(&gossipPacket)
        if err != nil {
          fmt.Println("Error encoding a simple message: ", err)
        }

        gossiper.Conn.WriteToUDP(packetBytes, udpAddr)
      }
    }
  }
}
