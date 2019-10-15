package structs

import "net"
import "fmt"
import "time"
import "strings"

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
  MyMessages *SeenMessages
  PacketChanel chan PacketAndAddresses
  MapOfChanels map[string]chan PacketAndAddresses
  MapHandler chan string
  MongeringMessages map[string]RumorMessage
  CurrentMessageID uint32
}

/*GossipPacket - To provide compatibility with future versions, the ONLY packets sent to other peers
    will be the GossipPacket's. For now it only contains a SimpleMessage*/
type GossipPacket struct {
  Simple  *SimpleMessage
  Rumor   *RumorMessage
  Status  *StatusPacket
}

//FlagsInformation A struct to hold flags information
type FlagsInformation struct {
  UIPort        string
  GossipAddress string
  Name          string
  Peers         string
  Simple        bool
  AntiEntropy   time.Duration
}

// ==================================================================
// ==================================================================
//                            Constructors
// ==================================================================
// ==================================================================


/*CreateNewGossiper - a function acting as a constructor for a the Gossiper struct, returns a pointer
  - address, string - the address for the gossiper node in the form 'ip:port'
  - flags, *FlagsInformation - a pointer to a FlagsInformation object
*/
func CreateNewGossiper(address string, flags *FlagsInformation) *Gossiper {
  udpAddr, err := net.ResolveUDPAddr("udp4", address)
  if err != nil {
    fmt.Println("Error resolving udp addres: ", err)
  }

  udpConn, err := net.ListenUDP("udp4", udpAddr)
  if err != nil {
    fmt.Println("Error listening: ", err)
  }

  peers := make(map[string]bool)
  for _, p := range (strings.Split(flags.Peers, ",")){
    peers[p] = true
  }

  var status []PeerStatus
  seenMessages := CreateSeenMessagesStruct()
  packetChanel := make(chan PacketAndAddresses)
  mapHandler := make(chan string)
  chanelMap := make(map[string]chan PacketAndAddresses)
  mongeringMap := make(map[string]RumorMessage)

  gossiper := &Gossiper{
    Address:      udpAddr,
    Conn:         udpConn,
    Name:         flags.Name,
    Peers:        peers,
    Want:         status,
    MyMessages:   seenMessages,
    PacketChanel: packetChanel,
    MapOfChanels: chanelMap,
    MapHandler:   mapHandler,
    MongeringMessages: mongeringMap,
    CurrentMessageID: 1,
  }

  fmt.Println("Gossiper is up and listening on ", address)
  return gossiper
}
