package main
import "flag"
import "net"
import "fmt"

func main() {

  var UIPortFlag = flag.String("UIPort", "8080", "port for the UI clinet")
  var msgFlag = flag.String("msg", "", "message to be sent")
  flag.Parse()

//  UIPort, _ := strconv.Atoi(*UIPort_flag)
  service := "127.0.0.1" + ":" + *UIPortFlag
  msg := *msgFlag

  // Opens a UDP connection to send messages read-in from the console
  remoteAddress, err := net.ResolveUDPAddr("udp4", service)
  if err != nil {
    fmt.Println("Error resolving udp address")
    return
  }
  udpConn, err := net.DialUDP("udp", nil, remoteAddress)
  if err != nil {
    fmt.Println("Error dialing udp")
    return
  }

  defer udpConn.Close()

  _, err = udpConn.Write([]byte(msg))
  if err != nil {
    fmt.Println("Error writing message")
    return
  }

  fmt.Println("MESSAGE $", msg, "$ was sent to ", service)

}
