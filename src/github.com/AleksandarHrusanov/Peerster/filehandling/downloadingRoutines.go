package filehandling

import (
	"bytes"
	"math/rand"
	"strings"
	"time"

	"github.com/AleksandarHrusanov/Peerster/core"
	"github.com/AleksandarHrusanov/Peerster/helpers"
)

func HandleClientImplicitDownloadRequest(gossiper *core.Gossiper, clientSearchRequest *core.Message) {
	var match *core.FileSearchMatch

	for _, searchMatch := range gossiper.OngoingFileSearch.MatchesFound {
		if searchMatch.ChunkCount == uint64(len(searchMatch.LocationOfChunks)) {
			if bytes.Compare(searchMatch.Metahash, *clientSearchRequest.Request) == 0 {
				match = searchMatch
				break
			}
		}
	}

	go initiateFileDownloading(gossiper, "", "", nil, match)
}

func initiateFileDownloading(gossiper *core.Gossiper, downloadFrom string, fname string, state *core.DownloadingState, fileMatch *core.FileSearchMatch) {

	if gossiper == nil {
		return
	}

	if fileMatch == nil {
		// we are issuing a regular file download
		if strings.Compare(downloadFrom, "") != 0 && strings.Compare(fname, "") != 0 && state != nil {
			initiateRegularDownloading(gossiper, downloadFrom, fname, state)
		}
	} else {
		// we are issuing a file download as a result from search request
		initiateFilesearchDownloading(gossiper, fileMatch)
	}
}

// TODO: Break this functoin into shorter functions (e.g 'handleReceivedMetafile')
func initiateFilesearchDownloading(gossiper *core.Gossiper, match *core.FileSearchMatch) {
	// downloadedFilesCount := 0
	// for _, match := range fileMatches {
	fname := match.FileName
	fInfo := &core.FileInformation{FileName: fname, ChunksMap: make(map[string][]byte)}
	gossiper.OngoingFileSearch.SearchRequestLock.Lock()
	ch := gossiper.OngoingFileSearch.SearchDownloadReplyChanel
	gossiper.OngoingFileSearch.SearchRequestLock.Unlock()

	// create a ticker
	ticker := time.NewTicker(5 * time.Second)
	haveMetaFile := false
	nextIdx := uint64(0)

	// pick a random peer to request the metafile from
	randomPeer := pickRandomPeerToRequestChunk(match, nextIdx)
	request := createDataRequest(gossiper.Name, randomPeer, match.Metahash)
	forwardDataRequest(gossiper, request)

	// in an infinite for-loop
	for {
		continueDownloading := true
		// select statement
		select {
		// if ticker timeout
		case <-ticker.C:
			// resend data request
			if !haveMetaFile {
				request := createDataRequest(gossiper.Name, randomPeer, match.Metahash)
				forwardDataRequest(gossiper, request)
			} else {
				downloadFrom := pickRandomPeerToRequestChunk(match, nextIdx)
				chunkHash := fInfo.Metafile[uint32(nextIdx)]
				helpers.PrintDownloadingChunk(fname, downloadFrom, uint32(nextIdx))
				resendDataRequest(gossiper, downloadFrom, chunkHash)
			}
		case reply := <-ch:
			if reply != nil {
				// if a dataReply comes from the chanel
				if len(reply.Data) == 0 {
					// if the Data field of the reply was empty, stop downloading
					continueDownloading = false
				} else {
					// sanity check - make sure it is a reply to my last request
					var lastChunkRequested []byte
					if !haveMetaFile {
						lastChunkRequested = match.Metahash
					} else {
						chunkHash := fInfo.Metafile[uint32(nextIdx)]
						lastChunkRequested = chunkHash[:]
					}
					if !replyWasExpected(lastChunkRequested, reply) {
						// received data reply for a chunk that was not requested; do nothing
					}
					if !replyIntegrityCheck(reply) {
						// received data reply with mismatching hash and data; resend request
						if !haveMetaFile {
							resendDataRequest(gossiper, randomPeer, convertSliceTo32Fixed(match.Metahash))
						} else {
							downloadFrom := pickRandomPeerToRequestChunk(match, nextIdx)
							chunkHash := fInfo.Metafile[uint32(nextIdx)]
							helpers.PrintDownloadingChunk(fname, downloadFrom, uint32(nextIdx))
							resendDataRequest(gossiper, downloadFrom, chunkHash)
						}
					} else {
						chunkHash := convertSliceTo32Fixed(reply.HashValue)
						chunkHashString := hashToString(chunkHash)
						if !haveMetaFile {
							fInfo.Metafile = mapifyMetafile(reply.Data)
							haveMetaFile = true
						} else {
							// the datareply SHOULD be containing a file data chunk
							// update FileInfo struct
							fInfo.ChunksMap[chunkHashString] = reply.Data[:len(reply.Data)]
						}

						// if that was the last chunk to be downloaded reconstruct save the full file
						if nextIdx == match.ChunkCount {
							helpers.PrintReconstructedFile(fname)
							continueDownloading = false
							reconstructAndSaveFullyDownloadedFile(fInfo)
						}
					}

					if continueDownloading {
						// if not, get next chunk request, (update ticker) and send it
						ticker = time.NewTicker(5 * time.Second)
						nextIdx++
						nextHashToRequest := fInfo.Metafile[uint32(nextIdx)]
						downloadFrom := pickRandomPeerToRequestChunk(match, nextIdx)
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
	// downloadedFilesCount++
	// if downloadedFilesCount == constants.FullMatchesThreshold {
	// 	// we stop search after FullMatchesThreshold found, so chances are very slim to go beyond that, but to make sure we check here
	// 	//		(e.g. finding more than one full match at the same time and one of those being the last one we need)
	// 	break
	// }
	// }
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

func pickRandomPeerToRequestChunk(match *core.FileSearchMatch, chunkIdx uint64) string {
	allChunkLocations := match.LocationOfChunks
	var peerSet []string
	rand.Seed(time.Now().UnixNano())

	if chunkIdx != 0 {
		peerSet = allChunkLocations[chunkIdx]
	} else {
		// pick a random peer to request metafile
		peerSet = allChunkLocations[(rand.Uint64() % match.ChunkCount)]
	}

	// pick a random peer to request chunk with id = chunkIdx
	randomPeerIdx := rand.Intn(len(peerSet))
	return peerSet[randomPeerIdx]
}
