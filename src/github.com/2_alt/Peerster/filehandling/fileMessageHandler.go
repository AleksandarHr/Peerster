package filehandling

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"
	// import "crypto/sha256"
	// import "encoding/hex"
	"github.com/2_alt/Peerster/core"
	"github.com/2_alt/Peerster/helpers"
	"github.com/dedis/protobuf"
)

func createDataRequest(origin string, dest string, hash []byte) *core.DataRequest {
	request := core.DataRequest{Origin: origin, Destination: dest, HopLimit: uint32(10), HashValue: hash[:]}
	return &request
}

func createDataReply(origin string, dest string, hash []byte, data []byte) *core.DataReply {
	reply := core.DataReply{Origin: origin, Destination: dest, HopLimit: uint32(10), HashValue: hash[:], Data: data}
	return &reply
}

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

func convertSliceTo32Fixed(slice []byte) [32]byte {
	var result [32]byte
	if len(slice) < 32 {
		copy(result[:], slice[:len(slice)])
	} else {
		copy(result[:], slice[:32])
	}
	return result
}

func convertSliceTo8192Fixed(slice []byte) [8192]byte {
	var result [8192]byte
	if len(slice) < 8192 {
		copy(result[:], slice[:len(slice)])
	} else {
		copy(result[:], slice[:8192])
	}
	return result
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
				if !replyIntegrityCheck(reply) {
					// received data reply with mismatching hash and data; resend request
					fmt.Println("-------------------- Integrity check FAILED")
					fmt.Println("Reply Hash = ", hashToString(convertSliceTo32Fixed(reply.HashValue)))
					fmt.Println("Data Hash = ", hashToString(computeSha256(reply.Data)))
					resendDataRequest(gossiper, downloadFrom)
				} else {
					fmt.Println("-------------------- Integrity check PASSED")
					if gossiper.DownloadingStates[downloadFrom].MetafileRequested &&
						!gossiper.DownloadingStates[downloadFrom].MetafileDownloaded {
						// the datareply SHOULD contain the metafile then
						fmt.Println("ooooooooooooooooooo We should get int here first!!!")
						handleReceivedMetafile(gossiper, reply)
					} else {
						// the datareply SHOULD be containing a file data chunk
						// update FileInfo struct
						fmt.Println("xxxxxxxxxxxxx We should not be getting here more than once")
						chunkHash := convertSliceTo32Fixed(reply.HashValue)
						chunkHashString := hashToString(chunkHash)
						chunkData := convertSliceTo8192Fixed(reply.Data)
						gossiper.DownloadingStates[downloadFrom].FileInfo.ChunksMap[chunkHashString] = chunkData
						gossiper.DownloadingStates[downloadFrom].NextChunkIndex++
					}
					// save chunk to a new file
					chunkHashValue := convertSliceTo32Fixed(reply.HashValue)
					chunkData := reply.Data
					chunkPath, _ := filepath.Abs(downloadedFilesFolder + hashToString(chunkHashValue))
					ioutil.WriteFile(chunkPath, chunkData, 0777)

					// if that was the last chunk to be downloaded close the chanel and save the full file
					if wasLastFileChunk(gossiper, reply) {
						fmt.Println("=============== Downloaded all the chunks. Time to reconstruct the file")
						gossiper.DownloadingStates[downloadFrom].DownloadFinished = true
						close(gossiper.DownloadingStates[downloadFrom].DownloadChanel)
						reconstructAndSaveFullyDownloadedFile(gossiper.DownloadingStates[downloadFrom].FileInfo)
						delete(gossiper.DownloadingStates, downloadFrom)
					} else {
						fmt.Println("++++++++++++++ Have more chunks to download. Request next chunk")
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

	// write metafile to file system
	path, _ := filepath.Abs(downloadedFilesFolder)
	metafilePath, _ := filepath.Abs(path + "/" + hashToString(convertSliceTo32Fixed(reply.HashValue)))
	ioutil.WriteFile(metafilePath, reply.Data, 0777)
}

func wasLastFileChunk(gossiper *core.Gossiper, reply *core.DataReply) bool {
	metafile := gossiper.DownloadingStates[reply.Origin].FileInfo.Metafile
	lastChunkInMetafile := gossiper.DownloadingStates[reply.Origin].FileInfo.Metafile[uint32(len(metafile)-1)]
	return bytes.Compare(reply.HashValue, lastChunkInMetafile[:]) == 0
}

// ===================================================================

// TODO:  Duplicated code -REFACTOR!!!
// A function to forward a data request to the corresponding next hop
func forwardDataRequest(gossiper *core.Gossiper, msg *core.DataRequest) {

	if msg.HopLimit == 0 {
		// if we have reached the HopLimit, drop the message
		return
	}

	forwardingAddress := gossiper.DestinationTable[msg.Destination]
	// If current node has no information about next hop to the destination in question
	if strings.Compare(forwardingAddress, "") == 0 {
		fmt.Println("NO FORWARDING ADDRESS AGAIN :??")
		// TODO: What to do if there is no 'next hop' known when peer has to forward a private packet
	}

	// Decrement the HopLimit right before forwarding the packet
	msg.HopLimit--
	// Encode and send packet
	packetToSend := core.GossipPacket{DataRequest: msg}
	packetBytes, err := protobuf.Encode(&packetToSend)
	helpers.HandleErrorFatal(err)
	core.ConnectAndSend(forwardingAddress, gossiper.Conn, packetBytes)
}

// A function to forward a data request to the corresponding next hop
func forwardDataReply(gossiper *core.Gossiper, msg *core.DataReply) {
	if msg.HopLimit == 0 {
		// if we have reached the HopLimit, drop the message
		return
	}

	forwardingAddress := gossiper.DestinationTable[msg.Destination]
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
	core.ConnectAndSend(forwardingAddress, gossiper.Conn, packetBytes)
}

// a function to resend the latest requested chunk
func resendDataRequest(gossiper *core.Gossiper, downloadFrom string) {
	if state, ok := gossiper.DownloadingStates[downloadFrom]; ok {
		chunkToRerequest := state.LatestRequestedChunk
		request := createDataRequest(gossiper.Name, downloadFrom, chunkToRerequest[:])
		forwardDataRequest(gossiper, request)
	}
}

// returns true if the received data reply corresponds to the latest requested hash
func replyWasExpected(requestHash []byte, reply *core.DataReply) bool {
	return bytes.Compare(requestHash, reply.HashValue) == 0
}

// returns true if the hash value in the reply corresponds to the data
func replyIntegrityCheck(reply *core.DataReply) bool {
	dataHash := computeSha256(reply.Data)
	actualHash := reply.HashValue
	return bytes.Compare(actualHash, dataHash[:]) == 0
}

func reconstructAndSaveFullyDownloadedFile(fileInfo *core.FileInformation) {
	// create a file
	fileData := make([]byte, 0)
	for _, chunk := range fileInfo.ChunksMap {
		// concatenate all file chunks into a single array
		fileData = append(fileData, chunk[:]...)
	}
	// create and write to file
	path, _ := filepath.Abs(downloadedFilesFolder)
	filePath, _ := filepath.Abs(path + "/" + fileInfo.FileName)
	ioutil.WriteFile(filePath, fileData[:], 0777)
}

func createDownloadingState(clientMsg *core.Message) *core.DownloadingState {
	downloadFrom := clientMsg.Destination
	fileName := clientMsg.File
	var requestedMetaHash [32]byte
	copy(requestedMetaHash[:], (*clientMsg.Request)[:32])
	fInfo := &core.FileInformation{FileName: *fileName, MetaHash: requestedMetaHash,
		Metafile: make(map[uint32][32]byte, 0), ChunksMap: make(map[string][8192]byte, 0)}
	state := core.DownloadingState{FileInfo: fInfo, DownloadFinished: false, MetafileDownloaded: false,
		MetafileRequested: true, NextChunkIndex: uint32(0), ChunksToRequest: make([][]byte, 0),
		DownloadingFrom: *downloadFrom, DownloadChanel: make(chan *core.DataReply)}
	return &state
}
