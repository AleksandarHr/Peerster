package core
import "net"
import "sync"
import "strings"
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
  FileInfo 	*FileInformation
  DownloadFinished bool
  NextChunkID uint32
  DownloadingFrom string
  lck sync.Mutex
}

// Gossiper Struct of a gossiper
type Gossiper struct {
	Address         	*net.UDPAddr
	Conn            	*net.UDPConn
	Name            	string
	KnownPeers      	[]string
	KnownRumors     	[]RumorMessage
	Want            	[]PeerStatus
	LocalAddr       	*net.UDPAddr
	LocalConn       	*net.UDPConn
	CurrentRumorID  	uint32
	MongeringStatus 	[]*MongeringStatus
	uiPort          	string
	DestinationTable	map[string]string
	DownloadingStates []*DownloadingState
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

	return &Gossiper{
		Address:         		udpAddr,
		Conn:            		udpConn,
		Name:            		name,
		KnownPeers:      		knownPeersList,
		KnownRumors:     		make([]RumorMessage, 0),
		Want:            		make([]PeerStatus, 0),
		LocalAddr:       		udpAddrLocal,
		LocalConn:       		udpConnLocal,
		CurrentRumorID:  		uint32(0),
		MongeringStatus: 		make([]*MongeringStatus, 0),
		uiPort:          		UIPort,
		DestinationTable: 	make(map[string]string),
		DownloadingStates:	make([]*DownloadingState, 0),
	}
}

// GetUIPort Get the gossiper's UI port
func (g *Gossiper) GetUIPort() string {
	return g.uiPort
}

// GetLocalAddr Get the gossiper's localConn as a string
func (g *Gossiper) GetLocalAddr() string {
	return g.LocalAddr.String()
}

// GetAllRumors Get the rumors known by the gossiper
func (g *Gossiper) GetAllRumors() []RumorMessage {
	return g.KnownRumors
}

// GetAllKnownPeers Get the known peers of this gossiper
func (g *Gossiper) GetAllKnownPeers() []string {
	return g.KnownPeers
}

// AddPeer Add a peer to the list of known peers
func (g *Gossiper) AddPeer(address string) {
	if helpers.IPAddressIsValid(address) {
		for _, peer := range g.KnownPeers {
			if strings.Compare(peer, address) == 0 {
				return
			}
		}
		g.KnownPeers = append(g.KnownPeers, address)
	}
}
