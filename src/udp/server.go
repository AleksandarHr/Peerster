package main
import "net"
import "fmt"

func main() {

  UDPConnection, _ := net.ListenUDP("udp", &net.UDPAddr{IP: []byte{0,0,0,0}, Port: 10001, Zone:""})
  defer UDPConnection.Close()

  buffer := make([]byte, 1024)
  for {
    numBytes, address, _ := UDPConnection.ReadFromUDP(buffer);
    fmt.Println("Message received: ", string(buffer[0:numBytes]), " from ", address)
  }

}
