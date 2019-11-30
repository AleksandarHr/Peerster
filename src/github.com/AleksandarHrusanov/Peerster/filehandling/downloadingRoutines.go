package filehandling

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/AleksandarHrusanov/Peerster/constants"
	"github.com/AleksandarHrusanov/Peerster/core"
	"github.com/AleksandarHrusanov/Peerster/helpers"
)

func initiateFileDownloading(gossiper *core.Gossiper, downloadFrom string, fname string, state *core.DownloadingState, fileMatches []*core.FileSearchMatch) {

	if gossiper == nil {
		return
	}

	if fileMatches == nil {
		// we are issuing a regular file download
		if strings.Compare(downloadFrom, "") != 0 && strings.Compare(fname, "") != 0 && state != nil {
			initiateRegularDownloading(gossiper, downloadFrom, fname, state)
		}
	} else {
		// we are issuing a file download as a result from search request
		initiateFilesearchDownloading(gossiper, fileMatches)
	}
}

// TODO: Break this functoin into shorter functions (e.g 'handleReceivedMetafile')
func initiateFilesearchDownloading(gossiper *core.Gossiper, fileMatches []*core.FileSearchMatch) {
	downloadedFilesCount := 0

	fmt.Println("Starting a filesearch download")
	for _, match := range fileMatches {
		fname := match.FileName
		fInfo := &core.FileInformation{FileName: fname, ChunksMap: make(map[string][]byte)}
		gossiper.OngoingFileSearch.SearchRequestLock.Lock()
		if !gossiper.OngoingFileSearch.IsOngoing {
			// Search is still ongoing, do not start downloading yet
			// return?
			fmt.Println("no ongoing file search, returning")
			gossiper.OngoingFileSearch.SearchRequestLock.Unlock()
			return
		}

		ch := gossiper.OngoingFileSearch.SearchDownloadReplyChanel
		gossiper.OngoingFileSearch.SearchRequestLock.Unlock()
		// currentMetafile := make(map[uint32][constants.HashSize]byte)

		// create a ticker
		ticker := time.NewTicker(5 * time.Second)
		nextIdx := uint64(0)
		// in an infinite for-loop

		rand.Seed(time.Now().UnixNano())
		randomPeer := match.LocationOfChunks[uint64(rand.Intn(len(match.LocationOfChunks)))]
		request := createDataRequest(gossiper.Name, randomPeer, match.Metahash)
		fmt.Println("Requesting metafile hash = " + hex.EncodeToString(match.Metahash))
		forwardDataRequest(gossiper, request)

		for {
			continueDownloading := true
			// select statement
			select {
			// if ticker timeout
			case <-ticker.C:
				// resend data request
				if nextIdx == 0 {
					request := createDataRequest(gossiper.Name, randomPeer, match.Metahash)
					forwardDataRequest(gossiper, request)
				} else {
					downloadFrom := match.LocationOfChunks[nextIdx]
					chunkHash := fInfo.Metafile[uint32(nextIdx)]
					helpers.PrintSearchDownloadingChunk(fname, downloadFrom, uint32(nextIdx))
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
						var lastChunk [constants.HashSize]byte
						if nextIdx == 0 {
							lastChunk = convertSliceTo32Fixed(match.Metahash)
						} else {
							lastChunk = fInfo.Metafile[uint32(nextIdx)]
						}
						if !replyWasExpected(lastChunk[:], reply) {
							// received data reply for a chunk that was not requested; do nothing
						}
						if !replyIntegrityCheck(reply) {
							// received data reply with mismatching hash and data; resend request
							if nextIdx == 0 {
								resendDataRequest(gossiper, randomPeer, convertSliceTo32Fixed(match.Metahash))
							} else {
								downloadFrom := match.LocationOfChunks[nextIdx]
								helpers.PrintSearchDownloadingChunk(fname, downloadFrom, uint32(nextIdx))
								chunkHash := fInfo.Metafile[uint32(nextIdx)]
								resendDataRequest(gossiper, downloadFrom, chunkHash)
							}
						} else {
							// the datareply SHOULD be containing a file data chunk
							// update FileInfo struct
							chunkHash := convertSliceTo32Fixed(reply.HashValue)
							chunkHashString := hashToString(chunkHash)
							if nextIdx == 0 {
								fInfo.Metafile = mapifyMetafile(reply.Data)
							} else {
								fInfo.ChunksMap[chunkHashString] = reply.Data[:len(reply.Data)]
							}
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
							nextHashToRequest := fInfo.Metafile[uint32(nextIdx)]
							downloadFrom := match.LocationOfChunks[nextIdx]
							request := createDataRequest(gossiper.Name, downloadFrom, nextHashToRequest[:])
							helpers.PrintSearchDownloadingChunk(fname, downloadFrom, uint32(nextIdx))
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
