package main
import "net"
import "bufio"
import "os"
import "fmt"

func main() {

  UDPConnection, _ := net.DialUDP("udp", nil, &net.UDPAddr{IP: []byte{127,0,0,1}, Port:10001, Zone:""})
  defer UDPConnection.Close()

  reader := bufio.NewReader(os.Stdin)
  for {
    fmt.Print("Enter text: ")
    text, _ := reader.ReadString('\n')
    if text == "Stop" {
      break
    }
    UDPConnection.Write([]byte(text))
  }


}
