package core

// Sha256HashSize - a constant for the size of a sha256 hash
const Sha256HashSize = uint32(32)

// FixedChunkSize - a constant for the size of a file chunk
const FixedChunkSize = uint32(8192)

// FileInformation - a structure to hold all information about a given file
type FileInformation struct {
	FileName      string
	NumberOfBytes uint32
	MetaHash      [32]byte
	Metafile      map[uint32][32]byte
	ChunksMap     map[string][8192]byte
	// HashedChunksMap map[string][]byte
}

// SimpleMessage simple message for part 1
type SimpleMessage struct {
	OriginalName  string
	RelayPeerAddr string
	Contents      string
}

// Message is sent between client and gossiper
type Message struct {
	Text        string
	Destination *string
	File        *string
	Request     *[]byte
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
	Origin      string
	ID          uint32
	Text        string
	Destination string
	HopLimit    uint32
}

// GossipPacket standard wrapper for communications
// between gossipers
type GossipPacket struct {
	Simple      *SimpleMessage
	Rumor       *RumorMessage
	Status      *StatusPacket
	Private     *PrivateMessage
	DataRequest *DataRequest
	DataReply   *DataReply
}

// DataRequest - a struct for requesting file chunks
type DataRequest struct {
	Origin      string
	Destination string
	HopLimit    uint32
	HashValue   []byte
}

// DataReply a struct for sending file chunks
type DataReply struct {
	Origin      string
	Destination string
	HopLimit    uint32
	HashValue   []byte
	Data        []byte
}
