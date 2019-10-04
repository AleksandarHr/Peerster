package main
import "flag"
import "net"
import "fmt"
import "strconv"
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
var localhostArr = []byte{127,0,0,1}
var mapOfPeers = make(map[string]bool)
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

type Gossiper struct {
  Address *net.UDPAddr
  Conn *net.UDPConn
  Name string
  Peers string
}

/*
  To provide compatibility with future versions, the ONLY packets sent to other peers
    will be the GossipPacket's. For now it only contains a SimpleMessage
*/
type GossipPacket struct {
  Simple *SimpleMessage
}

type FlagsInformation struct {
  UIPort string
  GossipAddress string
  Name string
  Peers string
  Simple bool
}

type IpPortPair struct {
  IP string
  Port string
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
    handlePeerMessages(gossiper, flags)
  }()
  wg.Wait()
}


func decoupleIpAndPort(ipPort string) IpPortPair {
  sl := strings.Split(ipPort, ":");
  pair := IpPortPair{IP : sl[0], Port : sl[1]}
  return pair;
}

func handleFlags() (*FlagsInformation) {
  var UIPort_flag = flag.String("UIPort", "8080", "port for the UI client")
  var gossipAddr_flag = flag.String("gossipAddr", localhost + ":" + "5000", "ip:port for the gossiper")
  var name_flag = flag.String("name", "new_node", "name of the gossiper")
  var peers_flag = flag.String("peers", "", "comma separated list of peers of the form ip:port")
  var simple_flag = flag.Bool("simple", true, "run gossiper in simple mode")
  // Parse all flagse
  flag.Parse()

  port := *UIPort_flag;
  gossipAddr := *gossipAddr_flag;
  name := *name_flag;
  peers := *peers_flag;
  simple := *simple_flag;

  flagsInfo := FlagsInformation{UIPort : port, GossipAddress : gossipAddr, Name : name, Peers : peers, Simple : simple}
  return &flagsInfo
}

func handleClientMessages(gossiper *Gossiper, uiPort string) {

  // Listen for incoming UDP messages from clients
  uiAddress := localhost + ":" + uiPort
  uiAddr, err := net.ResolveUDPAddr("udp4", uiAddress)
  if err != nil {
    fmt.Println("Error resolving udp addres: ", err)
  }

  uiConn, err := net.ListenUDP("udp4", uiAddr)
  if err != nil {
    fmt.Println("Error listening: ", err)
  }

  client_buffer := make([]byte, maxBufferSize)
  for {
    numBytes, conn, err := uiConn.ReadFromUDP(client_buffer)
    if err != nil {
      fmt.Println("could not read from UDP")
    }
    if conn == nil {
      continue
    }
    writeClientMessageToStandardOutput(string(client_buffer[0:numBytes]), gossiper.Peers)

    // Create a simple message object
    simpleMessage := SimpleMessage{}
    simpleMessage.OriginalName = gossiper.Name
    simpleMessage.RelayPeerAddr = string(gossiper.Address.IP) + string(gossiper.Address.Port)
    simpleMessage.Contents = string(client_buffer[0:numBytes])

    propagateGossipPacket(gossiper, simpleMessage)
  }
}

func handlePeerMessages (gossiper *Gossiper, flags *FlagsInformation) {

  // Goroutine (thread) to handle incoming messages from other gossipers
  //ipArray := strings.Split(ipPort.IP, ".")

  peer_buffer := make([]byte, maxBufferSize)
  for {
    numBytes, _, _ := gossiper.Conn.ReadFromUDP(peer_buffer)
    fmt.Println("NUMBER OF BYTES READ === ", numBytes)
    packet := SimpleMessage{}
    err := protobuf.Decode(peer_buffer[:numBytes], &packet)
    if err != nil {
      fmt.Println("Error decoding message: ", err)
    }
    writePeerMessageToStandardOutput(packet)
  }
}

func writeClientMessageToStandardOutput (msg, peers string) {
  fmt.Println("CLIENT MESSAGE " + msg)
  fmt.Println("PEERS " + peers)
}

func writePeerMessageToStandardOutput (msg SimpleMessage) {
  fmt.Println("SIMPLE MESSAGE origin " +
              msg.OriginalName +
              " from " +
              msg.RelayPeerAddr +
              " contents " + msg.Contents)
  fmt.Println("PEERS " + stringOfPeers.String())
}

func propagateGossipPacket (gossiper *Gossiper, msg SimpleMessage) {

  listOfPeers := strings.Split(gossiper.Peers, ",")

  for _, peer := range listOfPeers {
    pair := decoupleIpAndPort(peer)
    port, _ := strconv.Atoi(pair.Port)
    packetBytes, err := protobuf.Encode(&msg)
    if err != nil {
      fmt.Println("Error encoding a simple message: ", err)
    }
    fmt.Println("Number of bytes sent === ", len(packetBytes))
    gossiper.Conn.WriteToUDP(packetBytes, &net.UDPAddr{IP: []byte{127,0,0,1}, Port: port, Zone:""})
  }

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

  fmt.Println("Gossiper is up and listening on ", address)

  return &Gossiper {
    Address:  udpAddr,
    Conn:     udpConn,
    Name:     flags.Name,
    Peers:    flags.Peers,
  }
}
