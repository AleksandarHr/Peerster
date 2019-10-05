package main
import "flag"
import "net"
import "fmt"
//import "strconv"
import "strings"
import "sync"
import "github.com/dedis/protobuf"

/*
  Gossiper program
  Takes as arguments the following:
    - UIPort = string, port for the UI client (default "8080")
    - gossipAddr = string, ip:port for the gossiper (default "127.0.0.1:5000")
    - name = string, name of the gossiper
    - peers = A list of one or more addresses of other gossipers it knows of at the
      beginning, in the form 'ip1:port1,ip2:port2,...' -- NOTE: Use ports below 1024
    - simple = a flag to enforce simple broadcast mode for compatibility
*/

// When clients run locally, the gossiper listens for client on localhost
var maxBufferSize = 1024
var localhost = "127.0.0.1"
var stringOfPeers strings.Builder

/*
  To begin with, we will send 'simple messages' containing the following:
    - OriginalName = original sender's name
    - RelayPeerAddr = relay peer's address in the form 'ip:port'
    - Contents = the text message itself
*/
type SimpleMessage struct {
  OriginalName string
  RelayPeerAddr string
  Contents string
}

/*
  Gossiper struct containint
    * Address - the udp address of the gossiper node
    * Conn - the udp connection of the gossiper node
    * Name - the name of the gossiper node
    * Peers - a map of the addresses of peers' nodes known to this gossiper node
*/
type Gossiper struct {
  Address *net.UDPAddr
  Conn *net.UDPConn
  Name string
  Peers map[string]bool
}

/*
  To provide compatibility with future versions, the ONLY packets sent to other peers
    will be the GossipPacket's. For now it only contains a SimpleMessage
*/
type GossipPacket struct {
  Simple *SimpleMessage
}

// A struct to hold flags information
type FlagsInformation struct {
  UIPort string
  GossipAddress string
  Name string
  Peers string
  Simple bool
}

func main() {

  flags := handleFlags();
  gossiper := NewGossiper(flags.GossipAddress, flags);

  defer gossiper.Conn.Close();

  var wg = &sync.WaitGroup{}
  wg.Add(1)

  go func() {
    defer wg.Done()
    handleClientMessages(gossiper, flags.UIPort)
  } ()

  wg.Add(1)
  go func() {
    defer wg.Done()
    handlePeerMessages(gossiper)
  }()
  wg.Wait()
}

/*
  A function to handle flags passed to the gossiper as described at the beginning of the file
*/
func handleFlags() (*FlagsInformation) {
  // Read all the flags
  var UIPort_flag = flag.String("UIPort", "8080", "port for the UI client")
  var gossipAddr_flag = flag.String("gossipAddr", localhost + ":" + "5000", "ip:port for the gossiper")
  var name_flag = flag.String("name", "new_node", "name of the gossiper")
  var peers_flag = flag.String("peers", "", "comma separated list of peers of the form ip:port")
  var simple_flag = flag.Bool("simple", true, "run gossiper in simple mode")

  // Parse all flagse
  flag.Parse()

  // Save the flags information
  port := *UIPort_flag;
  gossipAddr := *gossipAddr_flag;
  name := *name_flag;
  peers := *peers_flag;
  simple := *simple_flag;

  flagsInfo := FlagsInformation{UIPort : port, GossipAddress : gossipAddr, Name : name, Peers : peers, Simple : simple}
  return &flagsInfo
}


/*
  A function to handle messages coming from a client
    * gossiper *Gossiper - poitner to a gossiper
    * uiPort string - the uiPort of the current gossiper
*/
func handleClientMessages(gossiper *Gossiper, uiPort string) {

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
  client_buffer := make([]byte, maxBufferSize)
  for {
    // Read incoming client message
    numBytes, conn, err := uiConn.ReadFromUDP(client_buffer)
    if err != nil {
      fmt.Println("could not read from UDP")
    }
    if conn == nil {
      continue
    }

    // Write received client message to standard output
    writeClientMessageToStandardOutput(gossiper, string(client_buffer[0:numBytes]))

    // Create a simple message object
    simpleMessage := SimpleMessage{}
    simpleMessage.OriginalName = gossiper.Name
    simpleMessage.RelayPeerAddr = gossiper.Address.String()
    simpleMessage.Contents = string(client_buffer[0:numBytes])
    gossipPacket := GossipPacket{Simple: &simpleMessage}

    // Propagate the client message in the form of a SimpleMessage to known peers
    propagateGossipPacket(gossiper, gossipPacket, "")
  }
}

/*
  A function to handle messages coming from a client
    * gossiper *Gossiper - poitner to a gossiper
*/
func handlePeerMessages (gossiper *Gossiper) {
  // Goroutine (thread) to handle incoming messages from other gossipers
  peer_buffer := make([]byte, maxBufferSize)
  for {
    numBytes, _, _ := gossiper.Conn.ReadFromUDP(peer_buffer)
    packet := GossipPacket{}
    err := protobuf.Decode(peer_buffer[:numBytes], &packet)
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

func writeClientMessageToStandardOutput (gossiper *Gossiper, msg string) {
  fmt.Println("CLIENT MESSAGE " + msg)
  fmt.Println("PEERS " + joinMapKeys(gossiper.Peers))
}

func writePeerMessageToStandardOutput (gossiper *Gossiper, packet GossipPacket) {
  fmt.Println("SIMPLE MESSAGE origin " +
              packet.Simple.OriginalName +
              " from " +
              packet.Simple.RelayPeerAddr +
              " contents " + packet.Simple.Contents)
  fmt.Println("PEERS " + joinMapKeys(gossiper.Peers))
}


func propagateGossipPacket (gossiper *Gossiper, gossipPacket GossipPacket, peerSenderAddress string) {

  peers := joinMapKeys(gossiper.Peers)

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

func joinMapKeys (m map[string]bool) string {

  keys := make([]string, 0, len(m))
  for k := range m {
    if k != "" {
      keys = append(keys, k)
    }
  }

  return strings.Join(keys, ",")
}

func NewGossiper(address string, flags *FlagsInformation) *Gossiper {
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

  return &Gossiper {
    Address:  udpAddr,
    Conn:     udpConn,
    Name:     flags.Name,
    Peers:    peers,
  }
}
