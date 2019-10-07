package helpers

import "fmt"
import "strings"
import "github.com/JohnDoe/Peerster/structs"


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
