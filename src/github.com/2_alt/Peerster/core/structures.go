package core

// SimpleMessage simple message for part 1
type SimpleMessage struct {
	OriginalName  string
	RelayPeerAddr string
	Contents      string
}

// Message is sent between client and gossiper
type Message struct {
	Text 				string
	Destination *string
	File				*string
	Request			*[]byte
}

// RumorMessage sent between gossipers
type RumorMessage struct {
	Origin string
	ID     uint32
	Text   string
}

// PeerStatus sent between gossipers
type PeerStatus struct {
	Identifier string
	NextID     uint32
}

// StatusPacket contains PeerStatus
type StatusPacket struct {
	Want []PeerStatus
}

// PrivateMessage contains a private message with a destination
type PrivateMessage struct {
	Origin			string
	ID					uint32
	Text				string
	Destination	string
	HopLimit		uint32
}

// GossipPacket standard wrapper for communications
// between gossipers
type GossipPacket struct {
	Simple 	*SimpleMessage
	Rumor  	*RumorMessage
	Status 	*StatusPacket
	Private *PrivateMessage
}
