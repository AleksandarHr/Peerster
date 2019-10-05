package helpers

import "net"
import "strings"

//FlagsInformation A struct to hold flags information
type FlagsInformation struct {
  UIPort string
  GossipAddress string
  Name string
  Peers string
  Simple bool
}

/*SimpleMessage - to begin with, we will send containing the following:
   - OriginalName = original sender's name
   - RelayPeerAddr = relay peer's address in the form 'ip:port'
   - Contents = the text message itself */
type SimpleMessage struct {
  OriginalName string
  RelayPeerAddr string
  Contents string
}

/*Gossiper - a struct containing
    * Address - the udp address of the gossiper node
    * Conn - the udp connection of the gossiper node
    * Name - the name of the gossiper node
    * Peers - a map of the addresses of peers' nodes known to this gossiper node*/
type Gossiper struct {
  Address *net.UDPAddr
  Conn *net.UDPConn
  Name string
  Peers map[string]bool
}

/*GossipPacket - To provide compatibility with future versions, the ONLY packets sent to other peers
    will be the GossipPacket's. For now it only contains a SimpleMessage*/
type GossipPacket struct {
  Simple *SimpleMessage
}

/*JoinMapKeys - a function which joins string map keys with comma */
func JoinMapKeys (m map[string]bool) string {

  keys := make([]string, 0, len(m))
  for k := range m {
    if k != "" {
      keys = append(keys, k)
    }
  }

  return strings.Join(keys, ",")
}
