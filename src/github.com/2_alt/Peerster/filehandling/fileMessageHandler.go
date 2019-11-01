package filehandling

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/2_alt/Peerster/core"
)

// HandlePeerDataReply - a function to handle data reply from other peers
func HandlePeerDataReply(gossiper *core.Gossiper, dataReply *core.DataReply) {
	origin := dataReply.Origin
	// if dataReplay message has reached destination
	if strings.Compare(dataReply.Destination, gossiper.Name) == 0 {
		// packet is for this gossiper
		// lock
		gossiper.DownloadingLock.Lock()
		if _, ok := gossiper.DownloadingStates[origin]; !ok {
			// there is not chanel for downloading from this origin, so we are not
			// in the process of downloading from them => Do nothing
		} else {
			// send to map of downloading states
			gossiper.DownloadingStates[origin].DownloadChanel <- dataReply
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
		retrievedChunk := retrieveRequestedHashFromFileSystem(convertSliceTo32Fixed(dataRequest.HashValue))
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

	downloadFrom := *clientMsg.Destination
	requestedMetaHash := *clientMsg.Request
	// We are initiating a new download with someone, asking for a metafile
	gossiper.DownloadingLock.Lock()
	if _, ok := gossiper.DownloadingStates[downloadFrom]; ok {
		// Already downloading from this peer - what to do?
	} else {
		// otherwise, create a new chanel for downloading from this peer
		newState := createDownloadingState(clientMsg)
		gossiper.DownloadingStates[downloadFrom] = newState
	}
	gossiper.DownloadingLock.Unlock()

	// create a DataRequest message and send a gossip packet
	request := createDataRequest(gossiper.Name, downloadFrom, requestedMetaHash)
	forwardDataRequest(gossiper, request)

	// start a new downloading go routine
	go initiateFileDownloading(gossiper, downloadFrom)
}

func initiateFileDownloading(gossiper *core.Gossiper, downloadFrom string) {
	// check if downloadFrom is present in the map (must be, we just created a chanel)
	gossiper.DownloadingLock.Lock()
	if _, ok := gossiper.DownloadingStates[downloadFrom]; !ok {
		// state for downloading from 'downloadFrom' node does not exists when it should
		// return?
		return
	}
	ch := gossiper.DownloadingStates[downloadFrom].DownloadChanel
	gossiper.DownloadingLock.Unlock()

	// create a ticker
	ticker := time.NewTicker(5 * time.Second)

	// in an infinite for-loop
	for {
		// select statement
		select {
		// if ticker timeout
		case <-ticker.C:
			// resend data request
			resendDataRequest(gossiper, downloadFrom)
		case reply := <-ch:
			if reply != nil {
				// TODO: SEE WHY USED TO RECEIVE NIL REPLIES
				// if a dataReply comes from the chanel
				// sanity check - make sure it is a reply to my last request
				if !replyWasExpected(gossiper.DownloadingStates[downloadFrom].LatestRequestedChunk[:], reply) {
					// received data reply for a chunk that was not requested; do nothing
				}
				fmt.Println("xxxxxxxxx Data length == ", len(reply.Data))
				fmt.Println("Next index is = ", gossiper.DownloadingStates[downloadFrom].NextChunkIndex)
				if !replyIntegrityCheck(reply) {
					// received data reply with mismatching hash and data; resend request
					fmt.Println("-------------------- Integrity check FAILED")
					resendDataRequest(gossiper, downloadFrom)
				} else {
					// fmt.Println("-------------------- Integrity check PASSED")
					if gossiper.DownloadingStates[downloadFrom].MetafileRequested &&
						!gossiper.DownloadingStates[downloadFrom].MetafileDownloaded {
						// the datareply SHOULD contain the metafile then
						handleReceivedMetafile(gossiper, reply)
					} else {
						// the datareply SHOULD be containing a file data chunk
						// update FileInfo struct
						fmt.Println("----------------------Received a chunk with bytes length = ", len(reply.Data))
						chunkHash := convertSliceTo32Fixed(reply.HashValue)
						chunkHashString := hashToString(chunkHash)
						chunkData := convertSliceTo8192Fixed(reply.Data)
						gossiper.DownloadingStates[downloadFrom].FileInfo.ChunksMap[chunkHashString] = chunkData
						gossiper.DownloadingStates[downloadFrom].NextChunkIndex++

						// save chunk to a new file
						chunkPath, _ := filepath.Abs(downloadedFilesFolder + chunkHashString)
						ioutil.WriteFile(chunkPath, chunkData[:], 0755)
					}

					// if that was the last chunk to be downloaded close the chanel and save the full file
					if wasLastFileChunk(gossiper, reply) {
						fmt.Println("=============== Downloaded all the chunks. Time to reconstruct the file")
						gossiper.DownloadingStates[downloadFrom].DownloadFinished = true
						close(gossiper.DownloadingStates[downloadFrom].DownloadChanel)
						reconstructAndSaveFullyDownloadedFile(gossiper.DownloadingStates[downloadFrom].FileInfo)
						delete(gossiper.DownloadingStates, downloadFrom)
					} else {
						// if not, get next chunk request, (update ticker) and send it
						ticker = time.NewTicker(5 * time.Second)
						nextChunkIdx := gossiper.DownloadingStates[downloadFrom].NextChunkIndex
						fmt.Println("Next chunk ID = ", nextChunkIdx)
						nextHashToRequest, _ := gossiper.DownloadingStates[downloadFrom].FileInfo.Metafile[nextChunkIdx]
						fmt.Println("Next hash to request = ", hashToString(nextHashToRequest))
						gossiper.DownloadingStates[downloadFrom].LatestRequestedChunk = nextHashToRequest
						request := createDataRequest(gossiper.Name, downloadFrom, nextHashToRequest[:])
						forwardDataRequest(gossiper, request)
					}
				}
			}
		}
	}
}

func handleReceivedMetafile(gossiper *core.Gossiper, reply *core.DataReply) {
	// read the metafile and populate hashedChunks in the file
	metafile := mapifyMetafile(reply.Data)
	gossiper.DownloadingStates[reply.Origin].FileInfo.Metafile = metafile
	gossiper.DownloadingStates[reply.Origin].MetafileDownloaded = true
	fmt.Println("Printing received metafile with hash == ", hashToString(computeSha256(reply.Data)))
	for i := 0; i < len(metafile); i++ {
		fmt.Println("Index = ", uint32(i), " ::: Hash = ", hashToString(metafile[uint32(i)]))
	}
	// write metafile to file system
	path, _ := filepath.Abs(downloadedFilesFolder)
	metafilePath, _ := filepath.Abs(path + "/" + hashToString(convertSliceTo32Fixed(reply.HashValue)))
	ioutil.WriteFile(metafilePath, reply.Data, 0755)
}

func wasLastFileChunk(gossiper *core.Gossiper, reply *core.DataReply) bool {
	metafile := gossiper.DownloadingStates[reply.Origin].FileInfo.Metafile
	lastChunkInMetafile := gossiper.DownloadingStates[reply.Origin].FileInfo.Metafile[uint32(len(metafile)-1)]
	return bytes.Compare(reply.HashValue, lastChunkInMetafile[:]) == 0
}
