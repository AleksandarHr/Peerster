package filehandling

import (
	"bytes"
	"encoding/hex"
	"strings"

	"github.com/AleksandarHrusanov/Peerster/core"
	"github.com/AleksandarHrusanov/Peerster/helpers"
)

// HandlePeerDataReply - a function to handle data reply from other peers
func HandlePeerDataReply(gossiper *core.Gossiper, dataReply *core.DataReply) {
	origin := dataReply.Origin
	hashValue := dataReply.HashValue
	// if dataReplay message has reached destination
	if strings.Compare(dataReply.Destination, gossiper.Name) == 0 {
		// packet is for this gossiper
		// lock
		gossiper.OngoingFileSearch.SearchRequestLock.Lock()
		// if there is a file search currently happening, this data reply might be for it,
		// so send it to the chanel for handling
		// if gossiper.OngoingFileSearch.IsOngoing {
		// fmt.Println("Sending the data reply to the search download chanel!!")
		gossiper.OngoingFileSearch.SearchDownloadReplyChanel <- dataReply
		// }
		gossiper.OngoingFileSearch.SearchRequestLock.Unlock()

		gossiper.DownloadingLock.Lock()
		if _, ok := gossiper.DownloadingStates[origin]; !ok {
			// there is not chanel for downloading from this origin, so we are not
			// in the process of downloading from them => Do nothing
		} else {
			// send to map of downloading states
			for _, download := range gossiper.DownloadingStates[origin] {
				if bytes.Compare(download.LatestRequestedChunk[:], hashValue) == 0 {
					download.DownloadChanel <- dataReply
				}
			}
		}
		// unlock
		gossiper.DownloadingLock.Unlock()
	} else {
		forwardDataReply(gossiper, dataReply)
	}
}

// HandlePeerDataRequest - a function to handle data requests from other peers
func HandlePeerDataRequest(gossiper *core.Gossiper, dataRequest *core.DataRequest) {
	// if dataRequest message has reached destination
	if strings.Compare(dataRequest.Destination, gossiper.Name) == 0 {
		// packet is for this gossiper
		// retrieve requested chunk/metafile from file system
		retrievedChunk := retrieveRequestedHashFromGossiperMemory(gossiper, convertSliceTo32Fixed(dataRequest.HashValue))
		if retrievedChunk == nil {
			// chunk was not found, do nothing
			return
		}
		// create a datareply object
		reply := createDataReply(gossiper.Name, dataRequest.Origin, dataRequest.HashValue, retrievedChunk)
		// send DataReply to the origin of the data request
		forwardDataReply(gossiper, reply)
	} else {
		forwardDataRequest(gossiper, dataRequest)
	}
}

// HandleClientDownloadRequest - a function to handle client request to download a file
func HandleClientDownloadRequest(gossiper *core.Gossiper, clientMsg *core.Message) {
	fname := *clientMsg.File
	downloadFrom := *clientMsg.Destination
	requestedMetaHash := *clientMsg.Request
	newState := createDownloadingState(clientMsg)
	// We are initiating a new download with someone, asking for a metafile
	gossiper.DownloadingLock.Lock()
	if _, ok := gossiper.DownloadingStates[downloadFrom]; ok {
		// Already downloading from this peer - what to do?
		gossiper.DownloadingStates[downloadFrom] = make([]*core.DownloadingState, 0)
	}
	gossiper.DownloadingStates[downloadFrom] = append(gossiper.DownloadingStates[downloadFrom], newState)
	gossiper.DownloadingLock.Unlock()

	// create a DataRequest message and send a gossip packet
	request := createDataRequest(gossiper.Name, downloadFrom, requestedMetaHash)
	helpers.PrintDownloadingMetafile(fname, downloadFrom)
	forwardDataRequest(gossiper, request)

	// start a new downloading go routine
	go initiateFileDownloading(gossiper, downloadFrom, fname, newState, nil)

	gossiper.FilesAndMetahashes.FilesLock.Lock()
	gossiper.FilesAndMetahashes.FileNamesToMetahashesMap[fname] = hex.EncodeToString(requestedMetaHash)
	gossiper.FilesAndMetahashes.FilesLock.Unlock()
}
