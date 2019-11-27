package filehandling

import (
	"bytes"
	"encoding/hex"
	"strings"
	"time"

	"github.com/AleksandarHrusanov/Peerster/constants"
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

func initiateFileDownloading(gossiper *core.Gossiper, downloadFrom string, fname string, state *core.DownloadingState, fileMatches []*core.FileSearchMatch) {
	if fileMatches == nil {
		initiateRegularDownloading(gossiper, downloadFrom, fname, state)
	} else {
		initiateFilesearchDownloading(gossiper, fileMatches)
	}
}

func getChunkHashByIndex(index uint64, metafileHash []byte) [constants.HashSize]byte {
	start := int((index - 1)) * constants.HashSize
	end := start + constants.HashSize
	return convertSliceTo32Fixed(metafileHash[start:end])
}

// TODO: Break this functoin into shorter functions (e.g 'handleReceivedMetafile')
func initiateFilesearchDownloading(gossiper *core.Gossiper, fileMatches []*core.FileSearchMatch) {
	downloadedFilesCount := 0

	for _, match := range fileMatches {
		fname := match.FileName
		fInfo := &core.FileInformation{FileName: fname, ChunksMap: make(map[string][]byte)}

		gossiper.OngoingFileSearch.SearchRequestLock.Lock()
		if gossiper.OngoingFileSearch.IsOngoing {
			// Search is still ongoing, do not start downloading yet
			// return?
			return
		}

		ch := gossiper.OngoingFileSearch.SearchDownloadReplyChanel

		// create a ticker
		ticker := time.NewTicker(5 * time.Second)
		nextIdx := uint64(1)
		// in an infinite for-loop
		for {
			continueDownloading := true
			// select statement
			select {
			// if ticker timeout
			case <-ticker.C:
				// resend data request
				downloadFrom := match.LocationOfChunks[nextIdx]
				chunkHash := getChunkHashByIndex(nextIdx, match.Metahash)
				helpers.PrintDownloadingChunk(fname, downloadFrom, uint32(nextIdx))
				resendDataRequest(gossiper, downloadFrom, chunkHash)
			case reply := <-ch:
				if reply != nil {
					// if a dataReply comes from the chanel
					if len(reply.Data) == 0 {
						// if the Data field of the reply was empty, stop downloading
						continueDownloading = false
					} else {
						// sanity check - make sure it is a reply to my last request
						lastChunk := getChunkHashByIndex(nextIdx, match.Metahash)
						if !replyWasExpected(lastChunk[:], reply) {
							// received data reply for a chunk that was not requested; do nothing
						}
						if !replyIntegrityCheck(reply) {
							// received data reply with mismatching hash and data; resend request
							downloadFrom := match.LocationOfChunks[nextIdx]
							helpers.PrintDownloadingChunk(fname, downloadFrom, uint32(nextIdx))
							chunkHash := getChunkHashByIndex(nextIdx, match.Metahash)
							resendDataRequest(gossiper, downloadFrom, chunkHash)
						} else {
							// the datareply SHOULD be containing a file data chunk
							// update FileInfo struct
							chunkHash := convertSliceTo32Fixed(reply.HashValue)
							chunkHashString := hashToString(chunkHash)
							fInfo.ChunksMap[chunkHashString] = reply.Data[:len(reply.Data)]
							nextIdx++

							// save chunk to a new file
							// chunkPath, _ := filepath.Abs(constants.DownloadedFilesChunksFolder + "/" + chunkHashString)
							// ioutil.WriteFile(chunkPath, reply.Data[:len(reply.Data)], constants.FileMode)
							// if that was the last chunk to be downloaded close the chanel and save the full file
							if nextIdx == match.ChunkCount {
								helpers.PrintReconstructedFile(fname)
								continueDownloading = false
								reconstructAndSaveFullyDownloadedFile(fInfo)
							}
						}

						if continueDownloading {
							// if not, get next chunk request, (update ticker) and send it
							ticker = time.NewTicker(5 * time.Second)
							nextHashToRequest := getChunkHashByIndex(nextIdx, match.Metahash)
							downloadFrom := match.LocationOfChunks[nextIdx]
							request := createDataRequest(gossiper.Name, downloadFrom, nextHashToRequest[:])
							helpers.PrintDownloadingChunk(fname, downloadFrom, uint32(nextIdx))
							forwardDataRequest(gossiper, request)
						}
					}
				}
			}
			if !continueDownloading {
				break
			}
		}
		downloadedFilesCount++
		if downloadedFilesCount == constants.FullMatchesThreshold {
			// we stop search after FullMatchesThreshold found, so chances are very slim to go beyond that, but to make sure we check here
			//		(e.g. finding more than one full match at the same time and one of those being the last one we need)
			gossiper.OngoingFileSearch.SearchRequestLock.Unlock()
			break
		}
	}
}

func initiateRegularDownloading(gossiper *core.Gossiper, downloadFrom string, fname string, state *core.DownloadingState) {
	// check if downloadFrom is present in the map (must be, we just created a chanel)
	gossiper.DownloadingLock.Lock()
	if _, ok := gossiper.DownloadingStates[downloadFrom]; !ok {
		// state for downloading from 'downloadFrom' node does not exists when it should
		// return?
		return
	}
	ch := state.DownloadChanel
	gossiper.DownloadingLock.Unlock()

	// create a ticker
	ticker := time.NewTicker(5 * time.Second)
	state.StateLock.Lock()
	// in an infinite for-loop
	for {
		continueDownloading := true
		// select statement
		select {
		// if ticker timeout
		case <-ticker.C:
			// resend data request
			nextIdx := state.NextChunkIndex
			helpers.PrintDownloadingChunk(fname, downloadFrom, nextIdx)

			resendDataRequest(gossiper, downloadFrom, state.LatestRequestedChunk)
		case reply := <-ch:
			if reply != nil {
				// if a dataReply comes from the chanel
				if len(reply.Data) == 0 {
					// if the Data field of the reply was empty, stop downloading
					continueDownloading = false
				} else {
					// sanity check - make sure it is a reply to my last request
					lastChunk := state.LatestRequestedChunk[:]
					if !replyWasExpected(lastChunk, reply) {
						// received data reply for a chunk that was not requested; do nothing
					}
					if !replyIntegrityCheck(reply) {
						// received data reply with mismatching hash and data; resend request
						nextIdx := state.NextChunkIndex
						helpers.PrintDownloadingChunk(fname, downloadFrom, nextIdx)
						resendDataRequest(gossiper, downloadFrom, state.LatestRequestedChunk)
					} else {
						mfReqeusted := state.MetafileRequested
						mfDownloaded := state.MetafileDownloaded
						if mfReqeusted && !mfDownloaded {
							// the datareply SHOULD contain the metafile then
							handleReceivedMetafile(gossiper, reply, fname, state)
						} else {
							// the datareply SHOULD be containing a file data chunk
							// update FileInfo struct
							chunkHash := convertSliceTo32Fixed(reply.HashValue)
							chunkHashString := hashToString(chunkHash)
							gossiper.DownloadingLock.Lock()
							state.FileInfo.ChunksMap[chunkHashString] = reply.Data[:len(reply.Data)]
							state.NextChunkIndex++
							gossiper.DownloadingLock.Unlock()

							// save chunk to a new file
							// chunkPath, _ := filepath.Abs(constants.DownloadedFilesChunksFolder + "/" + chunkHashString)
							// ioutil.WriteFile(chunkPath, reply.Data[:len(reply.Data)], constants.FileMode)
							// if that was the last chunk to be downloaded close the chanel and save the full file
							if wasLastFileChunk(gossiper, reply, state) {
								helpers.PrintReconstructedFile(fname)
								state.DownloadFinished = true
								continueDownloading = false
								reconstructAndSaveFullyDownloadedFile(state.FileInfo)
								// removeState(gossiper.DownloadingStates[downloadFrom])
								// delete(gossiper.DownloadingStates, downloadFrom)
							}
						}

						if continueDownloading {
							// if not, get next chunk request, (update ticker) and send it
							ticker = time.NewTicker(5 * time.Second)
							nextChunkIdx := state.NextChunkIndex
							nextHashToRequest, _ := state.FileInfo.Metafile[nextChunkIdx]
							state.LatestRequestedChunk = nextHashToRequest
							request := createDataRequest(gossiper.Name, downloadFrom, nextHashToRequest[:])
							helpers.PrintDownloadingChunk(fname, downloadFrom, state.NextChunkIndex)
							forwardDataRequest(gossiper, request)
						}
					}
				}
			}
		}
		if !continueDownloading {
			state.StateLock.Unlock()
			break
		}
	}
}

func handleReceivedMetafile(gossiper *core.Gossiper, reply *core.DataReply, fname string, state *core.DownloadingState) {
	// read the metafile and populate hashedChunks in the file
	metafile := mapifyMetafile(reply.Data)
	gossiper.DownloadingLock.Lock()
	state.FileInfo.Metafile = metafile
	state.MetafileDownloaded = true
	gossiper.DownloadingLock.Unlock()

	// write metafile to file system
	// metafilePath := buildChunkPath(constants.DownloadedFilesChunksFolder, reply.HashValue)
	// ioutil.WriteFile(metafilePath, reply.Data, constants.FileMode)
}
