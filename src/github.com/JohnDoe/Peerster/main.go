package main
import "flag"
import "net"
import "fmt"
import "strconv"
import "strings"
import "sync"
//import "github.com/dedis/protobuf"

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

/*
  To provide compatibility with future versions, the ONLY packets sent to other peers
    will be the GossipPacket's. For now it only contains a SimpleMessage
*/
type GossipPacket struct {
  Simple *SimpleMessage
}

type FlagsInformation struct {
  UIPort int
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

  var wg = &sync.WaitGroup{}
  wg.Add(1)

  go func() {
    defer wg.Done()
    handleClientMessages(flags)
  } ()

  wg.Add(1)
  go func() {
    defer wg.Done()
    handlePeerMessages(flags)
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

  port, _ := strconv.Atoi(*UIPort_flag);
  gossipAddr := *gossipAddr_flag;
  name := *name_flag;
  peers := *peers_flag;
  simple := *simple_flag;

  flagsInfo := FlagsInformation{UIPort : port, GossipAddress : gossipAddr, Name : name, Peers : peers, Simple : simple}
  return &flagsInfo
}

func handleClientMessages(flags *FlagsInformation) {
  // Listen for incoming UDP messages from clients
  client_buffer := make([]byte, maxBufferSize)
  UDPConnection, _ := net.ListenUDP("udp", &net.UDPAddr{IP: []byte{127,0,0,1}, Port: flags.UIPort, Zone:""})
  defer UDPConnection.Close()

  for {
    numBytes, _, _ := UDPConnection.ReadFromUDP(client_buffer)
    writeClientMessageToStandardOutput(string(client_buffer[0:numBytes]))

    // Create a simple message object
    simpleMessage := SimpleMessage{}
    simpleMessage.OriginalName = flags.Name
    simpleMessage.RelayPeerAddr = flags.GossipAddress
    simpleMessage.Contents = string(client_buffer)
  }
}

func handlePeerMessages (flags *FlagsInformation) {

  // Goroutine (thread) to handle incoming messages from other gossipers
  mapOfPeers[flags.GossipAddress] = true
  stringOfPeers.WriteString(flags.GossipAddress);

  ipPort := decoupleIpAndPort(flags.GossipAddress)
  port,_ := strconv.Atoi(ipPort.Port)
  //ipArray := strings.Split(ipPort.IP, ".")

  peer_buffer := make([]byte, maxBufferSize)
  UDPConnection, _ := net.ListenUDP("udp", &net.UDPAddr{IP: []byte{127,0,0,1}, Port: port, Zone:""})
  defer UDPConnection.Close()

  for {
    numBytes, address, _ := UDPConnection.ReadFromUDP(peer_buffer)
    fmt.Println("Message received: ", string(peer_buffer[0:numBytes]), " from ", address)

    // Create a simple message object
    simpleMessage := SimpleMessage{}
    simpleMessage.OriginalName = flags.Name
    simpleMessage.RelayPeerAddr = flags.GossipAddress
    simpleMessage.Contents = string(peer_buffer)
  }
}

func writeClientMessageToStandardOutput (msg string) {
  fmt.Println("CLIENT MESSAGE " + msg)
  fmt.Println("PEERS " + stringOfPeers.String())
}

func writePeerMessageToStandardOutput (msg SimpleMessage) {
  fmt.Println("SIMPLE MESSAGE origin " +
              msg.OriginalName +
              " from " +
              msg.RelayPeerAddr +
              " contents " + msg.Contents)
  fmt.Println("PEERS " + stringOfPeers.String())
}

func propagateGossipPacket (packet GossipPacket, senderAddress string) {
  if(len(senderAddress) > 0) {
    // Peer message - send to all known peers BUT DO NOT SEND BACK TO THE INITIAL SENDER
  } else {
    // Client message - simply send to all known peers
  }
}
