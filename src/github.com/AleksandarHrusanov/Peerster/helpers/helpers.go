package helpers

import "flag"
import "fmt"
import "time"
import "math/rand"
import "strings"
import "github.com/AleksandarHrusanov/Peerster/structs"

var localhost = "127.0.0.1"

/*HandleFlags - A function to handle flags passed to the gossiper as described at the beginning of the file
*/
func HandleFlags() (*structs.FlagsInformation) {
  // Read all the flags
  var UIPortFlag = flag.String("UIPort", "8080", "port for the UI client")
  var gossipAddrFlag = flag.String("gossipAddr", localhost + ":" + "5000", "ip:port for the gossiper")
  var nameFlag = flag.String("name", "new_node", "name of the gossiper")
  var peersFlag = flag.String("peers", "", "comma separated list of peers of the form ip:port")
  var simpleFlag = flag.Bool("simple", false, "run gossiper in simple mode")
  var antiEntropyDurationFlag = flag.Int("antiEntropy", 10, "duratino in seconds for anti entropy timeout")

  // Parse all flagse
  flag.Parse()

  // Save the flags information
  port := *UIPortFlag;
  gossipAddr := *gossipAddrFlag;
  name := *nameFlag;
  peers := *peersFlag;
  simple := *simpleFlag;
  antiEntropyDuration := *antiEntropyDurationFlag

  flagsInfo := structs.FlagsInformation{UIPort : port, GossipAddress : gossipAddr, Name : name, Peers : peers, Simple : simple, AntiEntropy: antiEntropyDuration}
  return &flagsInfo
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

/*AlreadySeenMessage - a function to check if a gossiper has already seen a certain message*/
func AlreadySeenMessage (gossiper *structs.Gossiper, rumor *structs.RumorMessage) bool {
  msgID := rumor.ID
  msgOrigin := rumor.Origin
  alreadySeen := false
  for _, msg := range gossiper.Want {
    if msg.Identifier == msgOrigin {
      alreadySeen = (msgID < msg.NextID)
    }
  }
  return alreadySeen
}

/*FlipCoin - a function to return 0 or 1*/
func FlipCoin() int {
  rand.Seed(time.Now().UnixNano())
  return rand.Intn(2)
}


/*ConvertPeerStatusVectorClockToMap - convert vector clock to map */
func ConvertPeerStatusVectorClockToMap (peerStatus []structs.PeerStatus) map[string]uint32{

  peerStatusMap := make(map[string]uint32)
  for _, p := range peerStatus {
    peerStatusMap[p.Identifier] = p.NextID
  }
  return peerStatusMap
}

/*ConvertPeerStatusMapToVectorClock - convert vector clock to map */
func ConvertPeerStatusMapToVectorClock (peerStatusMap map[string]uint32) []structs.PeerStatus {

  var peerStatusVector []structs.PeerStatus
  for k, v := range peerStatusMap {
    ps := structs.PeerStatus{Identifier: k, NextID: v}
    peerStatusVector = append(peerStatusVector, ps)
  }
  return peerStatusVector
}

// =======================================================================================
// =======================================================================================
//                    Functions to write to standard output
// =======================================================================================
// =======================================================================================

/*WriteToStandardOutputWhenClientMessageReceived - comment
*/
func WriteToStandardOutputWhenClientMessageReceived (gossiper *structs.Gossiper, msg string) {
  fmt.Println("CLIENT MESSAGE " + msg)
  fmt.Println("PEERS " + JoinMapKeys(gossiper.Peers))
}

/*WriteToStandardOutputWhenPeerSimpleMessageReceived - comment
*/
func WriteToStandardOutputWhenPeerSimpleMessageReceived (gossiper *structs.Gossiper, packet *structs.GossipPacket) {
  fmt.Println("SIMPLE MESSAGE origin " +
              packet.Simple.OriginalName +
              " from " +
              packet.Simple.RelayPeerAddr +
              " contents " + packet.Simple.Contents)
  fmt.Println("PEERS " + JoinMapKeys(gossiper.Peers))
}

/*WriteToStandardOutputWhenRumorMessageReceived - comment
*/
func WriteToStandardOutputWhenRumorMessageReceived (gossiper *structs.Gossiper, packet *structs.GossipPacket, senderAddress string) {
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
func WriteToStandardOutputWhenStatusMessageReceived (gossiper *structs.Gossiper, packet *structs.GossipPacket, senderAddress string) {

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
