package filehandling
import "strings"
import "github.com/2_alt/Peerster/core"

func handlePeerDataReply(gossiper *core.Gossiper, gossipPacket *core.GossipPacket) {

  // assume nil check outside of function
  dataReply := gossipPacket.DataReply
  // if dataReplay message has reached destination
  if strings.Compare(dataReply.Destination, gossiper.Name) == 0 {
    // packet is for this gossiper
    // lock
    // send to map of downloading states
    // unlock

    // if there was not entry in the map for this file/origin (this must be metafile then),
    //   create a new one - the other peer just wants to share a file with this one
    //   start requesting chunks
  } else {
    if dataReply.HopLimit == 0 {
      // packet is not for this gossiper and has reached hop limit, do nothing
    } else {
      // send to packet to next hop
    }
  }
}

func handlePeerDataRequest(gossiper *core.Gossiper, gossipPacket *core.GossipPacket){
  // assume nil check outside of function
  dataRequest := gossipPacket.DataRequest
  // if dataRequest message has reached destination
  if strings.Compare(dataRequest.Destination, gossiper.Name) == 0 {
    // packet is for this gossiper
    // retrieve requested chunk/metafile from file system
    // create a datareply object
    // send to the origin of the data request
  } else {
    if dataReply.HopLimit == 0 {
      // packet is not for this gossiper and has reached hop limit, do nothing
    } else {
      // send to packet to next hop
    }
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
