package structs

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
