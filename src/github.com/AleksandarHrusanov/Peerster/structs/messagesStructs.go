package structs

import "sync"

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

/*Message - a struct to hold a client message's text*/
type Message struct {
  Text string
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

/*SeenMessages - a struct to store messages that a gossiper has already seen
  It is practically a map where:
    * keys = origin names
    * values = arrays of seen rumor messages from this origin
*/
type SeenMessages struct {
  Messages map[string][]RumorMessage
  Lck sync.Mutex
}

/*PacketAndAddresses - a struct */
type PacketAndAddresses struct {
  Packet *GossipPacket
  SenderAddr string
  ReceiverAddr string
}

/*AddRemoveChanelFromMap - a struct */
type AddRemoveChanelFromMap struct {
  MapCh chan string
  Add bool
}

// ==================================================================
// ==================================================================
//                            Constructors
// ==================================================================
// ==================================================================

/*CreateNewSimpleMessage - a constructor for a simple message; returns a pointer to a SimpleMessage
*/
func CreateNewSimpleMessage(name string, relayAddress string, contents string) *SimpleMessage {
  sm := &SimpleMessage{OriginalName: name, RelayPeerAddr: relayAddress, Contents: contents}
  return sm
}

/*CreateNewRumorMessage - a constructor for a rumor message; returns a pointer to a RumorMessage
*/
func CreateNewRumorMessage(origin string, id uint32, text string) *RumorMessage {
  rm := &RumorMessage{Origin: origin, ID: id, Text: text}
  return rm
}

/*CreateNewPeerStatusPair - a constructor for a peer status message; returns a pointer to a PeerStatus
*/
func CreateNewPeerStatusPair(identifier string, nextID uint32) *PeerStatus {
  ps := &PeerStatus{Identifier: identifier, NextID: nextID}
  return ps
}

/*CreateSeenMessagesStruct - a constructor for a struct for seen rumor messages; returns a pointer to a SeenMessages
*/
func CreateSeenMessagesStruct() *SeenMessages{
  msgs := &SeenMessages{Messages: make(map[string][]RumorMessage)}
  return msgs
}

/*CreateNewStatusPacket - a constructor for a status packet; returns a pointer to a StatusPacket
*/
func CreateNewStatusPacket(peerStatusSlice []PeerStatus) *StatusPacket {
  sp := &StatusPacket{Want: peerStatusSlice}
  return sp
}
