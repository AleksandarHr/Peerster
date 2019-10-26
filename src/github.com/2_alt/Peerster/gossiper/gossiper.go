package gossiper

import (
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"
	"fmt"
	"github.com/2_alt/Peerster/core"
	"github.com/2_alt/Peerster/helpers"
	"github.com/dedis/protobuf"
)

// MongeringStatus struct of a mongering status linked with a timer
type MongeringStatus struct {
	rumorMessage          core.RumorMessage
	waitingStatusFromAddr string
	timeUp                chan bool
	ackReceived           bool
	lock                  sync.Mutex
}

// Gossiper Struct of a gossiper
type Gossiper struct {
	address         	*net.UDPAddr
	conn            	*net.UDPConn
	Name            	string
	knownPeers      	[]string
	knownRumors     	[]core.RumorMessage
	want            	[]core.PeerStatus
	localAddr       	*net.UDPAddr
	localConn       	*net.UDPConn
	currentRumorID  	uint32
	mongeringStatus 	[]*MongeringStatus
	uiPort          	string
	destinationTable	map[string]string
}

// GetUIPort Get the gossiper's UI port
func (g *Gossiper) GetUIPort() string {
	return g.uiPort
}

// GetLocalAddr Get the gossiper's localConn as a string
func (g *Gossiper) GetLocalAddr() string {
	return g.localAddr.String()
}

// GetAllRumors Get the rumors known by the gossiper
func (g *Gossiper) GetAllRumors() []core.RumorMessage {
	return g.knownRumors
}

// GetAllKnownPeers Get the known peers of this gossiper
func (g *Gossiper) GetAllKnownPeers() []string {
	return g.knownPeers
}

// AddPeer Add a peer to the list of known peers
func (g *Gossiper) AddPeer(address string) {
	if helpers.IPAddressIsValid(address) {
		for _, peer := range g.knownPeers {
			if strings.Compare(peer, address) == 0 {
				return
			}
		}
		g.knownPeers = append(g.knownPeers, address)
	}
}

// Retrieve a Rumor from a list given its Origin and ID
func getRumor(g *Gossiper, o string, i uint32) core.RumorMessage {
	for _, rm := range g.knownRumors {
		if strings.Compare(o, rm.Origin) == 0 && i == rm.ID {
			return rm
		}
	}
	return core.RumorMessage{}
}

// Update a slice of Rumor without duplicates
func updateRumorListNoDuplicates(r core.RumorMessage, list []core.RumorMessage) []core.RumorMessage {
	if len(list) > 0 {
		for _, rumorInList := range list {
			if strings.Compare(r.Origin, rumorInList.Origin) == 0 && rumorInList.ID == r.ID {
				return list
			}
		}
	}
	list = append(list, r)
	return list
}

// Update Want slice for given origin
func updateWant(g *Gossiper, origin string) {
	if strings.Compare(origin, "") == 0 {
		return
	}
	for i, peerStatus := range g.want {
		if strings.Compare(peerStatus.Identifier, origin) == 0 {
			g.want[i].NextID++
			return
		}
	}
	// Create a new PeerStatus if none exists
	newPeerStatus := core.PeerStatus{
		Identifier: origin,
		NextID:     uint32(2),
	}
	g.want = append(g.want, newPeerStatus)
}

// Send the status of the Gossiper to the given address
func sendStatus(gossiper *Gossiper, toAddr string) {
	if strings.Compare(toAddr, "") == 0 {
		panic("ERROR")
	}
	sp := core.StatusPacket{Want: gossiper.want}
	packetToSend := core.GossipPacket{Status: &sp}
	packetBytes, err := protobuf.Encode(&packetToSend)
	helpers.HandleErrorFatal(err)
	helpers.ConnectAndSend(toAddr, gossiper.conn, packetBytes)
}

// Send a RumorMessage to the given address
func sendRumor(r core.RumorMessage, gossiper *Gossiper, toAddr string) {
	packetToSend := core.GossipPacket{Rumor: &r}
	packetBytes, err := protobuf.Encode(&packetToSend)

	// Create new mongering status, set a timer for it and
	// append it to the slice in the gossiper struct
	newMongeringStatus := MongeringStatus{
		rumorMessage:          r,
		waitingStatusFromAddr: toAddr,
		timeUp:                make(chan bool),
		ackReceived:           false,
	}
	go func(mongeringStatusPtr *MongeringStatus) {
		time.Sleep(10 * time.Second)
		mongeringStatusPtr.timeUp <- true
	}(&newMongeringStatus)
	gossiper.mongeringStatus = append(gossiper.mongeringStatus, &newMongeringStatus)

	helpers.HandleErrorFatal(err)
	helpers.ConnectAndSend(toAddr, gossiper.conn, packetBytes)
}

// Get the gossiper current ID from its own Rumors. Useful when reconnecting
// to the network after having already sent some Rumors
func adjustMyCurrentID(g *Gossiper, status core.StatusPacket) {
	if g.currentRumorID == uint32(0) {
		currentMaxIDFromRumors := uint32(0)
		for _, st := range status.Want {
			if strings.Compare(st.Identifier, g.Name) == 0 && st.NextID > currentMaxIDFromRumors {
				currentMaxIDFromRumors = st.NextID
			}
		}
		if currentMaxIDFromRumors > 0 {
			g.currentRumorID = currentMaxIDFromRumors - 1
		}
	}
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
		address:         		udpAddr,
		conn:            		udpConn,
		Name:            		name,
		knownPeers:      		knownPeersList,
		knownRumors:     		make([]core.RumorMessage, 0),
		want:            		make([]core.PeerStatus, 0),
		localAddr:       		udpAddrLocal,
		localConn:       		udpConnLocal,
		currentRumorID:  		uint32(0),
		mongeringStatus: 		make([]*MongeringStatus, 0),
		uiPort:          		UIPort,
		destinationTable: 	make(map[string]string),
	}
}

// CreateSliceKnownPeers Create the slice of known peers necessary given a string containing
// all addresses
func CreateSliceKnownPeers(knownPeers string) []string {
	list := make([]string, 0)

	// No known-peers case
	if strings.Compare(knownPeers, "") == 0 {
		return list
	}

	// Fill the list
	s := ""
	for _, c := range knownPeers {
		if string(c) != "," {
			s += string(c)
		} else {
			list = append(list, s)
			s = ""
		}
	}
	list = append(list, s)

	return list
}

// Receive a message from UDP and decode it into a GossipPacket
func receiveAndDecode(gossiper *Gossiper) (core.GossipPacket, *net.UDPAddr) {
	// Create buffer
	buffer := make([]byte, 128)

	// Read message from UDP
	conn := gossiper.conn
	size, fromAddr, err := conn.ReadFromUDP(buffer)

	// Timeout
	helpers.HandleErrorNonFatal(err)
	if err != nil {
		return core.GossipPacket{}, nil
	}

	// Decode the packet
	gossipPacket := core.GossipPacket{}
	err = protobuf.Decode(buffer[0:size], &gossipPacket)
	helpers.HandleErrorNonFatal(err)

	return gossipPacket, fromAddr
}

// Receive a client's message from UDP and decode it into a GossipPacket
func receiveAndDecodeFromClient(gossiper *Gossiper) (core.Message, *net.UDPAddr) {
	// Create buffer
	buffer := make([]byte, 128)

	// Read message from UDP
	conn := gossiper.localConn
	size, fromAddr, err := conn.ReadFromUDP(buffer)

	// Timeout
	helpers.HandleErrorNonFatal(err)
	if err != nil {
		return core.Message{}, nil
	}

	// Decode the packet
	message := core.Message{}
	err = protobuf.Decode(buffer[0:size], &message)
	helpers.HandleErrorNonFatal(err)

	return message, fromAddr
}

// Repeat the rumor mongering process
func rumorMongerAgain(g *Gossiper) {
	// Send again the rumor to a different address
	rumorToSend := g.mongeringStatus[0].rumorMessage
	chosenAddr := helpers.PickRandomInSliceDifferentFrom(g.knownPeers,
		g.mongeringStatus[0].waitingStatusFromAddr)
	if strings.Compare(chosenAddr, "") != 0 {
		safeMongeringStatusDelete(g)
		sendRumor(rumorToSend, g, chosenAddr)
		helpers.PrintOutputMongering(chosenAddr)
	} else {
		safeMongeringStatusDelete(g)
	}
}

// Safe delete of mongering status
func safeMongeringStatusDelete(g *Gossiper) {
	statusToDelete := g.mongeringStatus[0]
	statusToDelete.lock.Lock()
	g.mongeringStatus = g.mongeringStatus[1:]
	statusToDelete.lock.Unlock()
}

// Remove all mongering status that timed-out and repeat the mongering process
func mongeringStatusRefresher(g *Gossiper) {
	for {
		if len(g.mongeringStatus) > 0 {
			if g.mongeringStatus[0] != nil {
				select {
				case <-g.mongeringStatus[0].timeUp:
					// Timed-out
					rumorMongerAgain(g)
				default:
					// Already acknowledged
					if g.mongeringStatus[0].ackReceived {
						safeMongeringStatusDelete(g)
					}
				}
			}
			// Wait
			time.Sleep(5 * time.Millisecond)
		} else {
			// Wait
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// Add a rumor to the gossiper's known rumors if it is not already there
func addRumorToKnownRumors(g *Gossiper, r core.RumorMessage) {
	for _, rumor := range g.knownRumors {
		if strings.Compare(rumor.Origin, r.Origin) == 0 && rumor.ID == r.ID {
			return
		}
	}
	g.knownRumors = append(g.knownRumors, r)
}

// Main peersListener function
func peersListener(gossiper *Gossiper, simpleMode bool) {
	// Remove all timed-out or handled mongering statuses
	go mongeringStatusRefresher(gossiper)

	// Infinite loop
	for {
		var gossipPacket core.GossipPacket
		var fromAddrPtr *net.UDPAddr
		fromAddr := ""

		// Listen
		gossipPacket, fromAddrPtr = receiveAndDecode(gossiper)
		if fromAddrPtr != nil {
			fromAddr = fromAddrPtr.String()
		}

		// Store address from the sender
		if !helpers.SliceContainsString(gossiper.knownPeers, fromAddr) &&
			strings.Compare(fromAddr, "") != 0 {
			gossiper.knownPeers = append(gossiper.knownPeers, fromAddr)
		}

		if gossipPacket.Simple != nil {
			// Print simple output
			simpleMessage := *gossipPacket.Simple
			helpers.PrintOutputSimpleMessageFromPeer(simpleMessage.Contents,
				simpleMessage.OriginalName,
				simpleMessage.RelayPeerAddr,
				gossiper.knownPeers)

			if simpleMode {
				// Prepare the message to be sent (SIMPLE MODE)
				simpleMessage.RelayPeerAddr = gossiper.address.String()
				packetToSend := core.GossipPacket{Simple: &simpleMessage}
				packetBytes, err := protobuf.Encode(&packetToSend)
				helpers.HandleErrorFatal(err)

				// Send message to all other known peers
				for _, knownAddress := range gossiper.knownPeers {
					if strings.Compare(knownAddress, fromAddr) != 0 {
						helpers.ConnectAndSend(knownAddress, gossiper.conn, packetBytes)
					}
				}
			}
		}

		if !simpleMode {
			if gossipPacket.Private != nil {
				// Handle incoming private message from another peer
				handlePrivateMessage(gossiper, gossipPacket.Private)
			}
			if gossipPacket.Rumor != nil {
				// Print RumorFromPeer output
				helpers.PrintOutputRumorFromPeer(gossipPacket.Rumor, fromAddr, gossiper.knownPeers)

				// Check if the Rumor or its Origin is known
				rumorIsKnown, originIsKnown, wantedID := helpers.IsRumorKnown(gossiper.want, gossipPacket.Rumor)

				if rumorIsKnown {
					// Do nothing

				} else if originIsKnown && wantedID == gossipPacket.Rumor.ID {
					// Update wantedID in Want slice
					updateWant(gossiper, gossipPacket.Rumor.Origin)

				} else if originIsKnown && wantedID < gossipPacket.Rumor.ID {
					// Do nothing

				} else if !originIsKnown {
					if gossipPacket.Rumor.ID == 1 {
						// If ID = 1, new Rumor: add it to list of known rumors
						// and create new PeerStatus
						updateWant(gossiper, gossipPacket.Rumor.Origin)

					} else {
						// If ID > 1, create new PeerStatus
						newPeerStatus := core.PeerStatus{
							Identifier: gossipPacket.Rumor.Origin,
							NextID:     uint32(1),
						}
						gossiper.want = append(gossiper.want, newPeerStatus)
					}
				}


				// Update destiantionTable
				helpers.UpdateDestinationTable(gossipPacket.Rumor.Origin, gossipPacket.Rumor.ID, fromAddr,
					gossiper.destinationTable, gossiper.knownRumors, originIsKnown, !helpers.IsRouteRumor(gossipPacket.Rumor))


				// Send status
				sendStatus(gossiper, fromAddr)

				if !rumorIsKnown {
					if len(gossiper.knownPeers) > 0 {
						// Pick a random address and send the rumor
						chosenAddr := helpers.PickRandomInSlice(gossiper.knownPeers)
						sendRumor(*gossipPacket.Rumor, gossiper, chosenAddr)
						helpers.PrintOutputMongering(chosenAddr)
					}
				}

				// Add Rumor to list of known Rumors if it is not already there
				addRumorToKnownRumors(gossiper, *gossipPacket.Rumor)
			} else if gossipPacket.Status != nil {
				// Print STATUS message
				helpers.PrintOutputStatus(fromAddr, gossipPacket.Status.Want, gossiper.knownPeers)

				// Check own rumorID to avoid crashes after reconnection (TODO)
				adjustMyCurrentID(gossiper, *gossipPacket.Status)

				// Check if the gossiper was waiting for this status packet and retrieve the
				// corresponding Rumor if we need to send it again after a coin flip
				rumorsToFlipCoinFor := make([]core.RumorMessage, 0)
				// For each mongeringstatus
				for _, mongeringStatus := range gossiper.mongeringStatus {
					// Check if the mongeringStatus has not yet been deleted
					if mongeringStatus != nil {
						mongeringStatus.lock.Lock()
					} else {
						continue
					}
					// Check that we were waiting an answer from this address
					if strings.Compare(mongeringStatus.waitingStatusFromAddr, fromAddr) == 0 {
						select {
						case _, ok := <-mongeringStatus.timeUp:
							// Do nothing if it has timed-up
							_ = ok
						default:
							// Check which Rumors have been acknowledge if any, can acknowldge
							// more than one Rumor
							if !mongeringStatus.ackReceived {
								// For each want of the other gossiper
								for _, want := range gossipPacket.Status.Want {
									if strings.Compare(want.Identifier, mongeringStatus.rumorMessage.Origin) == 0 &&
										want.NextID > mongeringStatus.rumorMessage.ID {
										rumorsToFlipCoinFor = updateRumorListNoDuplicates(mongeringStatus.rumorMessage,
											rumorsToFlipCoinFor)
										mongeringStatus.ackReceived = true
									}
								}
							}
						}
					}
					mongeringStatus.lock.Unlock()
				}

				// Check the received status packet
				iWantYourRumors := false
				youWantMyRumors := false
				var peerStatusTemp core.PeerStatus

				for _, peerStatus := range gossipPacket.Status.Want {
					peerFound := false
					for _, ownPeerStatus := range gossiper.want {
						if strings.Compare(peerStatus.Identifier, ownPeerStatus.Identifier) == 0 {
							peerFound = true
							if peerStatus.NextID < ownPeerStatus.NextID {
								// The other peer has not yet seen some of the Rumor I have
								youWantMyRumors = true
								peerStatusTemp = peerStatus
							} else if peerStatus.NextID > ownPeerStatus.NextID {
								// The other peer has some Rumor I do not have
								iWantYourRumors = true
							}
						}
					}
					// Case: the peer know a peer I do not know
					if !peerFound {
						iWantYourRumors = true
						newPeerStatus := core.PeerStatus{
							Identifier: peerStatus.Identifier,
							NextID:     1,
						}
						gossiper.want = append(gossiper.want, newPeerStatus)
					}
				}

				// Check if the other peer know the same peer as I do
				withFreshID := false
				if !iWantYourRumors && !youWantMyRumors {
					for _, ownPeerStatus := range gossiper.want {
						peerFound := false
						for _, peerStatus := range gossipPacket.Status.Want {
							if strings.Compare(peerStatus.Identifier, ownPeerStatus.Identifier) == 0 {
								peerFound = true
							}
						}
						// Case: I know a peer that the other peer don't
						if !peerFound {
							youWantMyRumors = true
							peerStatusTemp = ownPeerStatus
							withFreshID = true
						}
					}
				}

				// Solve the cases
				if youWantMyRumors {
					rumorToSend := core.RumorMessage{}
					if withFreshID {
						rumorToSend = getRumor(gossiper, peerStatusTemp.Identifier, 1)
					} else {
						rumorToSend = getRumor(gossiper, peerStatusTemp.Identifier, peerStatusTemp.NextID)
					}
					helpers.PrintOutputMongering(fromAddr)
					sendRumor(rumorToSend, gossiper, fromAddr)
				} else if iWantYourRumors {
					sendStatus(gossiper, fromAddr)
				} else {
					// Case: each peer is up-to-date and the gossiper
					// decide to continue rumormongering or not
					// Print IN SYNC WITH message
					helpers.PrintOutputInSyncWith(fromAddr)
					// Continue mongering test
					if len(rumorsToFlipCoinFor) > 0 {
						for _, rToFlip := range rumorsToFlipCoinFor {
							seed := rand.NewSource(time.Now().UnixNano())
							rng := rand.New(seed)
							flipCoinResult := rng.Intn(2)
							if flipCoinResult == 0 {
								// Pick a random address and send the rumor
								chosenAddr := helpers.PickRandomInSliceDifferentFrom(gossiper.knownPeers, fromAddr)
								if strings.Compare(chosenAddr, "") != 0 {
									// Print FLIPPED COIN message and send rumor
									sendRumor(rToFlip, gossiper, chosenAddr)
									helpers.PrintOutputFlippedCoin(chosenAddr)
								}
							}
						}
					}
				}
			}
		}
	}
}

func clientListener(gossiper *Gossiper, simpleMode bool) {
	for {
		// Receive and decode messages
		message, _ := receiveAndDecodeFromClient(gossiper)

		if simpleMode {
			// Print simple output
			helpers.PrintOutputSimpleMessageFromClient(message.Text, gossiper.knownPeers)

			// Prepare the message to be sent (SIMPLE MODE)
			simpleMessageToSend := core.SimpleMessage{
				OriginalName:  gossiper.Name,
				RelayPeerAddr: gossiper.address.String(),
				Contents:      message.Text,
			}
			packetToSend := core.GossipPacket{Simple: &simpleMessageToSend}
			packetBytes, err := protobuf.Encode(&packetToSend)
			helpers.HandleErrorFatal(err)

			// Send message to all known peers
			for _, knownAddress := range gossiper.knownPeers {
				helpers.ConnectAndSend(knownAddress, gossiper.conn, packetBytes)
			}
		}

		// Prepare the message to be sent
		if !simpleMode {

			if isClientMessagePrivate(&message) {
				// TODO: Handle private messages from client
				privateMsg := createNewPrivateMessage(gossiper.Name, message.Text, message.Destination)
				handlePrivateMessage(gossiper, privateMsg)
			} else {
				// Print output
				helpers.PrintOutputSimpleMessageFromClient(message.Text, gossiper.knownPeers)

				// Add rumor to list of known rumors
				gossiper.currentRumorID++
				newRumor := core.RumorMessage{
					Origin: gossiper.Name,
					ID:     gossiper.currentRumorID,
					Text:   message.Text,
				}
				addRumorToKnownRumors(gossiper, newRumor)
				updateWant(gossiper, gossiper.Name)

				// Pick a random address and send the rumor
				chosenAddr := ""
				if len(gossiper.knownPeers) > 0 {
					chosenAddr = helpers.PickRandomInSlice(gossiper.knownPeers)
					sendRumor(newRumor, gossiper, chosenAddr)
					helpers.PrintOutputMongering(chosenAddr)
				}
			}
		}
	}
}

// StartGossiper Start the gossiper
func StartGossiper(gossiperPtr *Gossiper, simplePtr *bool, antiEntropyPtr *int, routeRumorPtr *int) {
	rand.Seed(time.Now().UnixNano())

	// Listen from client and peers
	if !*simplePtr {
		go clientListener(gossiperPtr, *simplePtr)
		go peersListener(gossiperPtr, *simplePtr)
	} else {
		// In simple mode there is no anti-entropy so no infinite loop
		// to prevent the program to end
		go clientListener(gossiperPtr, *simplePtr)
		peersListener(gossiperPtr, *simplePtr)
	}
	defer gossiperPtr.conn.Close()
	defer gossiperPtr.localConn.Close()

	// Send the initial route rumor message on startup
	go routeRumorHandler(gossiperPtr, routeRumorPtr)

	// Anti-entropy
	if *antiEntropyPtr > 0 {
		for {
			time.Sleep(time.Duration(*antiEntropyPtr) * time.Second)

			if len(gossiperPtr.knownPeers) > 0 {
				randomAddress := helpers.PickRandomInSlice(gossiperPtr.knownPeers)
				sendStatus(gossiperPtr, randomAddress)
			}
		}
	} else {
		for {
			time.Sleep(time.Duration(*antiEntropyPtr) * 999)
		}
	}
}

// ========================================================
// ========================================================
//						Homework 2 functions
// ========================================================
// ========================================================

// A function to generate a route rumor (e.g. with empty Text field)
//		and send it to a randomly chosen known peer
func generateAndSendRouteRumor(gossiperPtr *Gossiper, rumorOrigin string, rumorID uint32) {
	chosenAddr := ""
	if len(gossiperPtr.knownPeers) > 0 {
		chosenAddr = helpers.PickRandomInSlice(gossiperPtr.knownPeers)
	}

	if strings.Compare(chosenAddr, "") != 0 {
		newRouteRumor := core.RumorMessage{
			Origin: rumorOrigin,
			ID:     rumorID,
			Text:		"",
		}
		packetToSend := core.GossipPacket{Rumor: &newRouteRumor}
		packetBytes, err := protobuf.Encode(&packetToSend)
		helpers.HandleErrorFatal(err)
		helpers.ConnectAndSend(chosenAddr, gossiperPtr.conn, packetBytes)
	}
}

// A function which sends the initial route rumor on start up and then sends
//		new route rumor periodically based on a user-specified flag
func routeRumorHandler(gossiperPtr *Gossiper, routeRumorPtr *int) {
	if *routeRumorPtr > 0 {
		// if the route rumor timer is 0, disable sending route rumors completely
		generateAndSendRouteRumor(gossiperPtr, gossiperPtr.Name, 1)
		for {
			time.Sleep(time.Duration(*routeRumorPtr) * time.Second)
			generateAndSendRouteRumor(gossiperPtr, gossiperPtr.Name, gossiperPtr.currentRumorID)
			gossiperPtr.currentRumorID++;
		}
	}
}

// Given a message from the client, return true if it is private
func isClientMessagePrivate(clientMsg *core.Message) bool {
	return (strings.Compare(*(clientMsg.Destination), "") != 0)
}

// A constructor for PrivateMessages - defaultID = 0 and defaultHopLimit = 10
func createNewPrivateMessage(origin string, msg string, dest *string) *core.PrivateMessage {
	defaultID := uint32(0) // to enforce NOT sequencing
	defaultHopLimit := uint32(10)
	privateMsg := core.PrivateMessage{Origin: origin, ID: defaultID, Text: msg, Destination: *dest, HopLimit: defaultHopLimit}
	return &privateMsg
}

func handlePrivateMessage(gossiper *Gossiper, privateMsg *core.PrivateMessage) {
	if privateMessageReachedDestination(gossiper, privateMsg) {
		// If private message reached its destination, print to console
		helpers.PrintOutputPrivateMessage(privateMsg.Origin, privateMsg.HopLimit, privateMsg.Text)
	} else {
		// If this is not the private message's destination, forward message to next hop
		forwardPrivateMessage(gossiper, privateMsg)
	}
}

// Given a private message, returns true if the current gossiper is it's destination
func privateMessageReachedDestination(gossiperPtr *Gossiper, msg *core.PrivateMessage) bool {
	return (strings.Compare(gossiperPtr.Name, msg.Destination) == 0)
}

// A function to forward a private message to the corresponding next hop
func forwardPrivateMessage(gossiperPtr *Gossiper, msg *core.PrivateMessage){

	if msg.HopLimit == 0 {
		// if we have reached the HopLimit, drop the message
		return
	}

	forwardingAddress := gossiperPtr.destinationTable[msg.Destination]
	// If current node has no information about next hop to the destination in question
	if strings.Compare(forwardingAddress, "") == 0 {
		// TODO: What to do if there is no 'next hop' known when peer has to forward a private packet
	}

	// Decrement the HopLimit right before forwarding the packet
	msg.HopLimit--
	// Encode and send packet
	packetToSend := core.GossipPacket{Private: msg}
	packetBytes, err := protobuf.Encode(&packetToSend)
	helpers.HandleErrorFatal(err)
	helpers.ConnectAndSend(forwardingAddress, gossiperPtr.conn, packetBytes)

}
