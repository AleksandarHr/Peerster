package helpers

import "net"
import "fmt"
import "strings"

//FlagsInformation A struct to hold flags information
type FlagsInformation struct {
  UIPort        string
  GossipAddress string
  Name          string
  Peers         string
  Simple        bool
}

/*SimpleMessage - to begin with, we will send containing the following:
   - OriginalName = original sender's name
   - RelayPeerAddr = relay peer's address in the form 'ip:port'
   - Contents = the text message itself */
type SimpleMessage struct {
  OriginalName  string
  RelayPeerAddr string
  Contents      string
}

/*RumorMessage - contains the actual text of a user message to be gossiperNode
    - Origin, string - identifies the message's original sender
    - ID, uint32     - contains the monotonically increasing sequence number
                       assigned by the original sender
    - Text, string   - the content of the message
*/
type RumorMessage struct {
  Origin  string
  ID      uint32
  Text    string
}

/*PeerStatus -
    - Identifier, string - origin's name
    - NextID, uint32     - the next unseen message sequence number
*/
type PeerStatus struct {
  Identifier  string
  NextID      uint32
}

/*StatusPacket - summarizes the set of messages the sending peer has seen so far
    - Want, []PeerStatus - a vector clock with origin IDs the peer knows about and
                           its associated values (uint32) represents the lowest
                           sequence number for which the peer has not yet seen a
                           message from the corresponding origin
*/
type StatusPacket struct {
  Want []PeerStatus
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
  Want []PeerStatus
}

/*GossipPacket - To provide compatibility with future versions, the ONLY packets sent to other peers
    will be the GossipPacket's. For now it only contains a SimpleMessage*/
type GossipPacket struct {
  Simple  *SimpleMessage
  Rumor   *RumorMessage
  Status  *StatusPacket
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


// =======================================================================================
// =======================================================================================
//                    Functions to write to standard output
// =======================================================================================
// =======================================================================================

/*WriteToStandardOutputWhenClientMessageReceived - comment
*/
func WriteToStandardOutputWhenClientMessageReceived (gossiper *Gossiper, msg string) {
  fmt.Println("CLIENT MESSAGE " + msg)
  fmt.Println("PEERS " + JoinMapKeys(gossiper.Peers))
}

/*WriteToStandardOutputWhenPeerSimpleMessageReceived - comment
*/
func WriteToStandardOutputWhenPeerSimpleMessageReceived (gossiper *Gossiper, packet *GossipPacket) {
  fmt.Println("SIMPLE MESSAGE origin " +
              packet.Simple.OriginalName +
              " from " +
              packet.Simple.RelayPeerAddr +
              " contents " + packet.Simple.Contents)
  fmt.Println("PEERS " + JoinMapKeys(gossiper.Peers))
}

/*WriteToStandardOutputWhenRumorMessageReceived - comment
*/
func WriteToStandardOutputWhenRumorMessageReceived (gossiper *Gossiper, packet *GossipPacket, senderAddress string) {
  fmt.Println("RUMOR origin " +
              packet.Rumor.Origin +
              " from " +
              senderAddress +
              " ID " +
              fmt.Sprint(packet.Rumor.ID) +
              " contents " +
              packet.Rumor.Text)
  fmt.Println("PEERS " + JoinMapKeys(gossiper.Peers))
}

/*WriteToStandardOutputWhenMongering - comment
*/
func WriteToStandardOutputWhenMongering (peerAddress string) {
  fmt.Println("MONGERING with " + peerAddress)
}

/*WriteToStandardOutputWhenStatusMessageReceived - comment
*/
func WriteToStandardOutputWhenStatusMessageReceived (gossiper *Gossiper, packet *GossipPacket, senderAddress string) {

  var statusString strings.Builder
  for _, pair := range packet.Status.Want {
    statusString.WriteString(" peer " + pair.Identifier + " nextID " + fmt.Sprint(pair.NextID))
  }

  fmt.Println("STATUS from " +
              senderAddress +
              statusString.String())
}

/*WriteToStandardOutputWhenFlippedCoin - comment
*/
func WriteToStandardOutputWhenFlippedCoin (chosenPeerAddress string) {
  fmt.Println("FLIPPED COIN sending rumor to " + chosenPeerAddress)
}

/*WriteToStandardOutputWhenInSyncWith - comment
*/
func WriteToStandardOutputWhenInSyncWith (senderAddress string) {
  fmt.Println("IN SYNC WITH " + senderAddress)
}
