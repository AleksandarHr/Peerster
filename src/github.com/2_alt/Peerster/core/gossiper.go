package core

import "net"
import "sync"

import "github.com/2_alt/Peerster/helpers"

// MongeringStatus struct of a mongering status linked with a timer
type MongeringStatus struct {
	RumorMessage          RumorMessage
	WaitingStatusFromAddr string
	TimeUp                chan bool
	AckReceived           bool
	Lock                  sync.Mutex
}

// DownloadingState - a struct of a file downloading state
type DownloadingState struct {
	FileInfo             *FileInformation
	DownloadFinished     bool
	MetafileDownloaded   bool
	MetafileRequested    bool
	NextChunkIndex       uint32
	LatestRequestedChunk [32]byte
	ChunksToRequest      [][]byte
	DownloadingFrom      string
	DownloadChanel       chan *DataReply
	StateLock            sync.Mutex
}

// SafeDestinationTable - a struct for the DSDV with a lock
type SafeDestinationTable struct {
	Dsdv     map[string]string
	DsdvLock sync.Mutex
}

// Gossiper Struct of a gossiper
type Gossiper struct {
	Address           *net.UDPAddr
	Conn              *net.UDPConn
	Name              string
	KnownPeers        []string
	KnownRumors       []RumorMessage
	Want              []PeerStatus
	LocalAddr         *net.UDPAddr
	LocalConn         *net.UDPConn
	CurrentRumorID    uint32
	RumorIDLock       sync.Mutex
	MongeringStatus   []*MongeringStatus
	uiPort            string
	DestinationTable  *SafeDestinationTable
	DownloadingStates map[string][]*DownloadingState
	DownloadingLock   sync.Mutex
}

// NewGossiper Create a new Gossiper
func NewGossiper(address string, name string,
	knownPeersList []string, UIPort string) *Gossiper {
	udpAddr, err := net.ResolveUDPAddr("udp4", address)
	helpers.HandleErrorFatal(err)
	udpConn, err := net.ListenUDP("udp4", udpAddr)
	helpers.HandleErrorFatal(err)
	clientAddr := "127.0.0.1:" + UIPort
	udpAddrLocal, err := net.ResolveUDPAddr("udp4", clientAddr)
	helpers.HandleErrorFatal(err)
	udpConnLocal, err := net.ListenUDP("udp4", udpAddrLocal)
	helpers.HandleErrorFatal(err)
	dsdv := &SafeDestinationTable{Dsdv: make(map[string]string)}

	return &Gossiper{
		Address:           udpAddr,
		Conn:              udpConn,
		Name:              name,
		KnownPeers:        knownPeersList,
		KnownRumors:       make([]RumorMessage, 0),
		Want:              make([]PeerStatus, 0),
		LocalAddr:         udpAddrLocal,
		LocalConn:         udpConnLocal,
		CurrentRumorID:    uint32(0),
		MongeringStatus:   make([]*MongeringStatus, 0),
		uiPort:            UIPort,
		DestinationTable:  dsdv,
		DownloadingStates: make(map[string][]*DownloadingState),
	}
}
