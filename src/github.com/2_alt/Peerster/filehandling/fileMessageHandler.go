package filehandling
import "strings"
import "github.com/2_alt/Peerster/core"
import "github.com/2_alt/Peerster/helpers"
import "github.com/dedis/protobuf"


func createDataRequest(origin string, dest string, hash [core.Sha256HashSize]byte) *core.DataRequest{
  request := core.DataRequest{Origin: origin, Destination: dest, HopLimit: uint32(10), HashValue: hash[:]}
  return &request
}

func createDataReply(origin string, dest string, hash [core.Sha256HashSize]byte, data []byte) *core.DataRequest{
  reply := core.DataReply{Origin: origin, Destination: dest, HopLimit: uint32(10), HashValue: hash[:], Data: data}
  return &reply
}

func handlePeerDataReply(gossiper *core.Gossiper, gossipPacket *core.GossipPacket) {
  // assume nil check outside of function
  dataReply := gossipPacket.DataReply
  origin := dataReply.Origin
  // if dataReplay message has reached destination
  if strings.Compare(dataReply.Destination, gossiper.Name) == 0 {
    // packet is for this gossiper
    // lock
    gossiper.DownloadingStates.DownloadingLock.Lock()
    if ch, ok := gossiper.DownloadingStates[origin]; !ok {
      // if there was not entry in the map for this file/origin (this must be metafile then),
      //   create a new one - the other peer just wants to share a file with this one
      //   start requesting chunks
    } else {
      // send to map of downloading states
      gossiper.DownloadingStates[origin] <- dataReply
    }
    
    // unlock
    gossiper.DownloadingStates.DownloadingLock.Unlock()
  } else {
    forwardDataReply(gossiper, dataReply)
  }
}

func handlePeerDataRequest(gossiper *core.Gossiper, gossipPacket *core.GossipPacket){
  // assume nil check outside of function
  dataRequest := gossipPacket.DataRequest
  // if dataRequest message has reached destination
  if strings.Compare(dataRequest.Destination, gossiper.Name) == 0 {
    // packet is for this gossiper
    // retrieve requested chunk/metafile from file system
    retrievedChunk := retrieveRequestedHashFromFileSystem(dataRequest.HashValue)

    // create a datareply object
    reply := createDataReply(gossiper.Name, dataRequest.Origin, dataRequest.HashValue, retrievedChunk)
    packetToSend := core.GossipPacket{DataReply: reply}
    // send to the origin of the data request

  } else {
    forwardDataRequest(gossiper, dataRequest)
  }
}

// TODO: UNCLEAR????????
func handleClientShareRequest(gossiper *core.Gossiper, clientMsg *core.Message) {
  // a client message which has a file specified to index, divide, and hash
  //    as well as destination where to start sending the file

  // handle file indexing
  // call filesharing functionality
}

func handleClientDownloadRequest(gossiper *core.Gossiper, clientMsg *core.Message){
  // We are initiating a new download with someone

  // check downloading map - if already downloading, do nothing
  // otherwise, create a new chanel for downloading from this peer
  // create a DataRequest message and send a gossip packet

  // start a new downloading go routine
}


func initiateFileDownloading(gossiper *core.Gossiper, downloadFrom string) {
  // check if downloadFrom is present in the map (must be, we just created a chanel)

  // create a ticker
  // in an infinite for-loop
  // select statement
      // if ticker timeout, resend data request
      // if a dataReply comes from the chanel
          // sanity check - make sure it is a reply to my last request (HOW???)
          // if that was the last chunk to be downloaded close the chanel
          // if not, get next chunk request, (update ticker) and send it
}


// ===================================================================
// Duplicated code - deal with it!!!

// A function to forward a data request to the corresponding next hop
func forwardDataRequest(gossiperPtr *core.Gossiper, msg *core.DataRequest){

	if msg.HopLimit == 0 {
		// if we have reached the HopLimit, drop the message
		return
	}

	forwardingAddress := gossiperPtr.DestinationTable[msg.Destination]
	// If current node has no information about next hop to the destination in question
	if strings.Compare(forwardingAddress, "") == 0 {
		// TODO: What to do if there is no 'next hop' known when peer has to forward a private packet
	}

	// Decrement the HopLimit right before forwarding the packet
	msg.HopLimit--
	// Encode and send packet
	packetToSend := core.GossipPacket{DataRequest: msg}
	packetBytes, err := protobuf.Encode(&packetToSend)
	helpers.HandleErrorFatal(err)
	core.ConnectAndSend(forwardingAddress, gossiperPtr.Conn, packetBytes)
}

// A function to forward a data request to the corresponding next hop
func forwardDataReply(gossiperPtr *core.Gossiper, msg *core.DataReply){

	if msg.HopLimit == 0 {
		// if we have reached the HopLimit, drop the message
		return
	}

	forwardingAddress := gossiperPtr.DestinationTable[msg.Destination]
	// If current node has no information about next hop to the destination in question
	if strings.Compare(forwardingAddress, "") == 0 {
		// TODO: What to do if there is no 'next hop' known when peer has to forward a private packet
	}

	// Decrement the HopLimit right before forwarding the packet
	msg.HopLimit--
	// Encode and send packet
	packetToSend := core.GossipPacket{DataReply: msg}
	packetBytes, err := protobuf.Encode(&packetToSend)
	helpers.HandleErrorFatal(err)
	core.ConnectAndSend(forwardingAddress, gossiperPtr.Conn, packetBytes)
}
