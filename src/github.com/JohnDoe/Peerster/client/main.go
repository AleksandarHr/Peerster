package main
import "flag"
import "net"
import "strconv"

func main() {

  var UIPort_flag = flag.String("UIPort", "8080", "port for the UI clinet")
  var msg_flag = flag.String("msg", "", "message to be sent")
  flag.Parse()

  UIPort, _ := strconv.Atoi(*UIPort_flag)
  msg := *msg_flag

  // Opens a UDP connection to send messages read-in from the console
  UDPConnection, _ := net.DialUDP("udp", nil, &net.UDPAddr{IP: []byte{127,0,0,1}, Port:UIPort, Zone:""})
  defer UDPConnection.Close()
  UDPConnection.Write([]byte(msg))
}
