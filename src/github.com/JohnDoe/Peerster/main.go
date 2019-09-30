package main
import "flag"
import "net"
import "fmt"
import "strconv"

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
var localhost = "127.0.0.1"
var localhostArr = []byte{127,0,0,1}

func main() {

  var UIPort_flag = flag.String("UIPort", "8080", "port for the UI client")
  var gossipAddr_flag = flag.String("gossipAddr", localhost + ":" + "5000", "ip:port for the gossiper")
  var name_flag = flag.String("name", "new_node", "name of the gossiper")
  var peers_flag = flag.String("peers", "", "comma separated list of peers of the form ip:port")
  var simple_flag = flag.Bool("simple", true, "run gossiper in simple mode")
  // Parse all flagse
  flag.Parse()

  UIPort, _ := strconv.Atoi(*UIPort_flag);
  gossipAddr := *gossipAddr_flag;
  name := *name_flag;
  peers := *peers_flag;
  simple := *simple_flag;

  fmt.Println("UIPort = ", UIPort)
  fmt.Println("GossipAddr = ", gossipAddr)
  fmt.Println("Name = ", name)
  fmt.Println("Peeres = ", peers)
  fmt.Println("Simple = ", simple)

  // Listen for incoming UDP messages
  UDPConnection, _ := net.ListenUDP("udp", &net.UDPAddr{IP: []byte{0,0,0,0}, Port: UIPort, Zone:""})
  defer UDPConnection.Close()

  buffer := make([]byte, 1024)
  for {
    //go func() {
      // Goroutine (thread) to handle incoming messages from other peers
    //}

    //go func() {
      // Goroutine (thread) to handle incoming messages from client
    //}
    numBytes, address, _ := UDPConnection.ReadFromUDP(buffer);
    fmt.Println("Message received: ", string(buffer[0:numBytes]), " from ", address)
  }
}
