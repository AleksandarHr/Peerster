package core

import (
	"net"
	"sync"

	"github.com/AleksandarHrusanov/Peerster/constants"
	"github.com/AleksandarHrusanov/Peerster/helpers"
)

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
	LatestRequestedChunk [constants.HashSize]byte
	ChunksToRequest      [][]byte
	DownloadingFrom      string
	DownloadChanel       chan *DataReply
	StateLock            sync.Mutex
}

type ChunkedDownloadingState struct {
	FileToDownload       *FileSearchMatch
	DownloadFinished     bool
	NextChunkIndex       uint32
	LatestRequestedChunk [constants.HashSize]byte
	LatestRequestedFrom  string
	StateLock            sync.Mutex
	DownloadChanel       chan *DataReply
}

// SafeDestinationTable - a struct for the DSDV with a lock
type SafeDestinationTable struct {
	Dsdv     map[string]string
	DsdvLock sync.Mutex
}

//SafeFilesAndMetahashes - a struct to hold names and metahahes of shared files
type SafeFilesAndMetahashes struct {
	MetaStringToFileInfo     map[string]*FileInformation
	FileNamesToMetahashesMap map[string]string
	MetaHashes               map[string][]byte
	AllChunks                map[string][]byte
	FilesLock                sync.Mutex
}

//SafeDownloadingStates - a struct to hold downloading states
type SafeDownloadingStates struct {
	DownloadingStates map[string][]*DownloadingState
	DownloadingLock   sync.Mutex
}

//SafePrivateMessages - a struct to hold private message exchanges
type SafePrivateMessages struct {
	Messages    map[string][]string
	MessageLock sync.Mutex
}

type SafeRecentFileSearches struct {
	Searches     map[string]bool
	SearchesLock sync.Mutex
}

// Gossiper Struct of a gossiper
// TODO: Change MongeringStatus to a map for faster access
type Gossiper struct {
	Address            *net.UDPAddr
	Conn               *net.UDPConn
	uiPort             string
	LocalAddr          *net.UDPAddr
	LocalConn          *net.UDPConn
	Name               string
	KnownPeers         []string
	PeersLock          sync.Mutex
	KnownRumors        []RumorMessage
	CurrentRumorID     uint32
	RumorIDLock        sync.Mutex
	Want               []PeerStatus
	MongeringStatus    []*MongeringStatus
	DestinationTable   *SafeDestinationTable
	PrivateMessages    *SafePrivateMessages
	FilesAndMetahashes *SafeFilesAndMetahashes
	DownloadingStates  map[string][]*DownloadingState
	DownloadingLock    sync.Mutex
	OngoingFileSearch  *SafeOngoingFileSearching
	RecentSearches     *SafeRecentFileSearches
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
	filesAndMetahashes := &SafeFilesAndMetahashes{FileNamesToMetahashesMap: make(map[string]string),
		MetaStringToFileInfo: make(map[string]*FileInformation),
		AllChunks:            make(map[string][]byte), MetaHashes: make(map[string][]byte)}
	privateMessages := &SafePrivateMessages{Messages: make(map[string][]string)}
	recentSearches := &SafeRecentFileSearches{Searches: make(map[string]bool)}
	ongoingSearch := CreateSafeOngoingFileSearching()

	return &Gossiper{
		Address:            udpAddr,
		Conn:               udpConn,
		Name:               name,
		KnownPeers:         knownPeersList,
		KnownRumors:        make([]RumorMessage, 0),
		Want:               make([]PeerStatus, 0),
		LocalAddr:          udpAddrLocal,
		LocalConn:          udpConnLocal,
		CurrentRumorID:     uint32(0),
		MongeringStatus:    make([]*MongeringStatus, 0),
		uiPort:             UIPort,
		DestinationTable:   dsdv,
		PrivateMessages:    privateMessages,
		FilesAndMetahashes: filesAndMetahashes,
		DownloadingStates:  make(map[string][]*DownloadingState),
		RecentSearches:     recentSearches,
		OngoingFileSearch:  ongoingSearch,
	}
}
