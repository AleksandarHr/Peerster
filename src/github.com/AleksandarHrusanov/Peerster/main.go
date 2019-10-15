package main
//import "strconv"
import "sync"
import "github.com/AleksandarHrusanov/Peerster/gossiper"
import "github.com/AleksandarHrusanov/Peerster/helpers"
import "github.com/AleksandarHrusanov/Peerster/structs"

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


func main() {

  flags := helpers.HandleFlags();
  gossiperNode := structs.CreateNewGossiper(flags.GossipAddress, flags);
  defer gossiperNode.Conn.Close();

  var wg = &sync.WaitGroup{}
  wg.Add(1)
  go func() {
    defer wg.Done()
    gossiper.HandleClientMessages(gossiperNode, flags.UIPort, flags.Simple)
  } ()

  wg.Add(1)
  go func() {
    defer wg.Done()
    gossiper.HandleGossipPackets(gossiperNode, flags.Simple, gossiperNode.PacketChanel)
  }()

  wg.Add(1)
  go func() {
    defer wg.Done()
    gossiper.HandleAntiEntropy(gossiperNode, flags.AntiEntropy)
  }()

  wg.Add(1)
  go func() {
    defer wg.Done()
    gossiper.HandleChanelMap(gossiperNode, gossiperNode.MapHandler)
  }()

  wg.Wait()

}
