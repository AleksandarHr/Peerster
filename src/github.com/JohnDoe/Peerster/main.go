package main
//import "strconv"
import "sync"
import "github.com/JohnDoe/Peerster/gossiper"

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

func main() {

  flags := gossiper.HandleFlags();
  gossiperNode := gossiper.NewGossiper(flags.GossipAddress, flags);

  defer gossiperNode.Conn.Close();

  var wg = &sync.WaitGroup{}
  wg.Add(1)

  go func() {
    defer wg.Done()
    gossiper.HandleClientMessages(gossiperNode, flags.UIPort)
  } ()

  wg.Add(1)
  go func() {
    defer wg.Done()
    gossiper.HandlePeerMessages(gossiperNode)
  }()
  wg.Wait()
}
