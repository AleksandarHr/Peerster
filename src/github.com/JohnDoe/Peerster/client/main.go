package main
import "flag"
import "net"
import "strconv"

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

func main() {

  var UIPort_flag = flag.String("UIPort", "8080", "port for the UI clinet")
  var msg_flag = flag.String("msg", "", "message to be sent")
  flag.Parse()

  UIPort, _ := strconv.Atoi(*UIPort_flag)
  msg := *msg_flag

  // Opens a UDP connection to send messages read-in from the console
  UDPConnection, _ := net.DialUDP("udp", nil, &net.UDPAddr{IP: []byte{127,0,0,1}, Port:UIPort, Zone:""})
  defer UDPConnection.Close()

  for {
    UDPConnection.Write([]byte(msg))
  }
}
