package filehandling

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
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

// HandlePeerDataRequest - a function to handle data requests from other peers
func HandlePeerDataRequest(gossiper *core.Gossiper, dataRequest *core.DataRequest) {
	// if dataRequest message has reached destination
	if strings.Compare(dataRequest.Destination, gossiper.Name) == 0 {
		// packet is for this gossiper
		// retrieve requested chunk/metafile from file system
		retrievedChunk := retrieveRequestedHashFromFileSystem(dataRequest.HashValue)
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
			// if a dataReply comes from the chanel
			// sanity check - make sure it is a reply to my last request (HOW???)
			if !replyWasExpected(gossiper.DownloadingStates[downloadFrom].LatestRequestedChunk, reply) {
				// received data reply for a chunk that was not requested; do nothing
			}
			if !replyIntegrityCheck(reply) {
				// received data reply with mismatching hash and data; resend request
				resendDataRequest(gossiper, downloadFrom)
			} else {
				if gossiper.DownloadingStates[downloadFrom].MetafileRequested &&
					!gossiper.DownloadingStates[downloadFrom].MetafileDownloaded {
					// the datareply SHOULD contain the metafile then
					handleReceivedMetafile(gossiper, reply)
				} else {
					// the datareply SHOULD be containing a file data chunk
					// update FileInfo struct
					chunkHash := reply.HashValue
					chunkHashString := hashToString(chunkHash)
					chunkData := reply.Data
					chunkIdx := gossiper.DownloadingStates[downloadFrom].NextChunkIndex
					gossiper.DownloadingStates[downloadFrom].FileInfo.HashedChunksMap[chunkHashString] = chunkHash
					gossiper.DownloadingStates[downloadFrom].FileInfo.ChunksMap[chunkIdx] = chunkData
					gossiper.DownloadingStates[downloadFrom].NextChunkIndex++
				}
				// save chunk to a new file
				chunkHashValue := reply.HashValue
				chunkData := reply.Data
				chunkPath, _ := filepath.Abs(downloadedFilesFolder + hashToString(chunkHashValue))
				file, err := os.Open(chunkPath)
				if err != nil {
					fmt.Println("Error opening a file: ", err)
				}
				ioutil.WriteFile(chunkPath, chunkData, 0777)
				file.Close()

				// if that was the last chunk to be downloaded close the chanel
				if wasLastFileChunk(gossiper, reply) {
					gossiper.DownloadingStates[downloadFrom].DownloadFinished = true
					reconstructAndSaveFullyDownloadedFile(gossiper.DownloadingStates[downloadFrom].FileInfo)
				} else {
					// if not, get next chunk request, (update ticker) and send it
					ticker = time.NewTicker(5 * time.Second)
					// nextRequest := *core.DataRequest{}
					// gossiper.DownloadingStates[downloadFrom].LatestRequestedChunk = nextRequest
					// request := createDataRequest(gossiper.Name, downloadFrom, nextRequest)
					// forwardDataRequest(gossiper, request)
				}
			}
		}
	}
}

func handleReceivedMetafile(gossiper *core.Gossiper, reply *core.DataReply) {
	gossiper.DownloadingStates[reply.Origin].FileInfo.Metafile = reply.Data
	gossiper.DownloadingStates[reply.Origin].MetafileDownloaded = true

	// read the metafile and populate hashedChunks in the file
	for i := 0; i < len(reply.Data); i += 32 {
		if len(reply.Data) <= i+32 {
			gossiper.DownloadingStates[reply.Origin].FileInfo.ChunksMap[uint32(i/32)] = reply.Data[i:len(reply.Data)]
		} else {
			gossiper.DownloadingStates[reply.Origin].FileInfo.ChunksMap[uint32(i/32)] = reply.Data[i : i+32]
		}
	}
}

func wasLastFileChunk(gossiper *core.Gossiper, reply *core.DataReply) bool {
	return false
}

// // TODO: UNCLEAR???????? DOESNT EXIST???
// func handleClientShareRequest(gossiper *core.Gossiper, clientMsg *core.Message) {
//   // a client message which has a file specified to index, divide, and hash
//   //    as well as destination where to start sending the file
//
//   // handle file indexing
//   // call filesharing functionality
// }

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
	chunkToRerequest := gossiper.DownloadingStates[downloadFrom].LatestRequestedChunk
	request := createDataRequest(gossiper.Name, downloadFrom, chunkToRerequest)
	forwardDataRequest(gossiper, request)
}

// returns true if the received data reply corresponds to the latest requested hash
func replyWasExpected(requestHash []byte, reply *core.DataReply) bool {
	return bytes.Compare(requestHash, reply.HashValue) == 0
}

// returns true if the hash value in the reply corresponds to the data
func replyIntegrityCheck(reply *core.DataReply) bool {
	dataHash := computeSha256(reply.Data)
	actualHash := reply.HashValue
	return bytes.Compare(actualHash, dataHash) == 0
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
	requestedMetaHash := clientMsg.Request
	fInfo := &core.FileInformation{FileName: *fileName, MetaHash: *requestedMetaHash}
	state := core.DownloadingState{FileInfo: fInfo, DownloadFinished: false, MetafileDownloaded: false,
		MetafileRequested: false, NextChunkIndex: uint32(0), ChunksToRequest: make([][]byte, 0),
		DownloadingFrom: *downloadFrom, DownloadChanel: make(chan *core.DataReply)}
	return &state
}
