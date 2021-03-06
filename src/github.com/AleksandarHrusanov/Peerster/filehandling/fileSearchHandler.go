package filehandling

import (
	"encoding/hex"
	"regexp"
	"strings"
	"time"

	"github.com/AleksandarHrusanov/Peerster/constants"
	"github.com/AleksandarHrusanov/Peerster/core"
	"github.com/AleksandarHrusanov/Peerster/helpers"
	"github.com/dedis/protobuf"
)

// A function to handle a search request coming from the client of this peerster node
func HandleClientSearchRequest(gossiper *core.Gossiper, clientSearchRequest *core.Message) {
	// start a new search request - declare and initialize an SafeOngoingFileSearching variable
	// fileSearch := core.CreateSafeOngoingFileSearching(clientSearchRequest.Budget, clientSearchRequest.Keywords)

	// save it in the gossipere
	// gossiper.OngoingFileSearch = fileSearch
	// fire a new thread to handle this
	go initiateFileSearching(gossiper, clientSearchRequest.Budget, clientSearchRequest.Keywords)
}

// A function to handle a search request coming from another peerster node
func HandlePeerSearchRequest(gossiper *core.Gossiper, searchRequest *core.SearchRequest) {
	// If the origin of the search request is not the this peer node itself
	// 1) Detect and discard duplicate requests (e.g. same Origin and same Keywords) in the last 0.5 seconds
	if strings.Compare(searchRequest.Origin, gossiper.Name) != 0 {
		searchSignature := append([]string{searchRequest.Origin}, searchRequest.Keywords...)
		duplicateSignature := strings.Join(searchSignature, ",")

		//  store somewhere/somehow recent search requests - check if contains this
		gossiper.RecentSearches.SearchesLock.Lock()
		if _, searched := gossiper.RecentSearches.Searches[duplicateSignature]; searched {
			//      if so, return
			gossiper.RecentSearches.SearchesLock.Unlock()
			return
		}
		//      otherwise, update recent search requests and TIME 0.5 seconds (go routine which
		//      deletes the message from the recent search requests after 0.5 seconds?)
		gossiper.RecentSearches.Searches[duplicateSignature] = true
		go func() {
			time.Sleep(500 * time.Millisecond)
			gossiper.RecentSearches.SearchesLock.Lock()
			delete(gossiper.RecentSearches.Searches, duplicateSignature)
			gossiper.RecentSearches.SearchesLock.Unlock()
		}()
		gossiper.RecentSearches.SearchesLock.Unlock()

		// 2) process the search request locally (and possibly send a SearchReply)
		//   Check both _SharedFiles and _Downloades (gossiper memory) for keywords matches
		searchResults := performLocalFilenameSearch(gossiper, searchRequest.Keywords)
		//   If any matches found, create and send a Search Reply (using next hop?)
		if len(searchResults) > 0 {
			searchReply := &core.SearchReply{Origin: gossiper.Name, Destination: searchRequest.Origin, HopLimit: uint32(10), Results: searchResults}
			forwardSearchReply(gossiper, searchReply)
		}
	}
	// 4) subtract 1 from the request's budget
	searchRequest.Budget--
	if searchRequest.Budget <= 0 {
		//    4.1) If remaining budget is <= 0, search failed, do nothing, return
		return
	}
	//    5) If the search request was issued by the gossiper, perform ring-expand - redistribute the remainig
	//         budget as evenly as possible to up to B neighboring nodes
	//         every peer gets a search request with budget = integer part of (budget / #neighbours)
	//         and then iteratively add 1 to the budget of the first R neighbors (where R = budget % # neighbors)
	gossiper.PeersLock.Lock()
	peers := gossiper.KnownPeers
	gossiper.PeersLock.Unlock()
	peersCount := uint64(len(peers))

	if peersCount > 0 {
		baseBdg := searchRequest.Budget / peersCount
		extraBdg := searchRequest.Budget % peersCount
		for _, peer := range peers {
			newBdg := uint64(baseBdg)
			if extraBdg > 0 {
				newBdg++
				extraBdg--
			}
			newSearchRequest := &core.SearchRequest{Origin: searchRequest.Origin, Budget: newBdg, Keywords: searchRequest.Keywords}
			packetToSend := core.GossipPacket{SearchRequest: newSearchRequest}
			packetBytes, err := protobuf.Encode(&packetToSend)
			helpers.HandleErrorFatal(err)
			core.ConnectAndSend(peer, gossiper.Conn, packetBytes)
		}
	}
}

// A function to handle a search reply coming from another peerster node
func HandlePeerSearchReply(gossiper *core.Gossiper, searchReply *core.SearchReply) {
	// NOTE: assume all search replies correspond to previously-issued search requests
	// 1) if destination field is not the current gossiper's name, forward with hop limit
	if strings.Compare(gossiper.Name, searchReply.Destination) != 0 {
		forwardSearchReply(gossiper, searchReply)
		return
	}
	// 2) if current node was the destination of the serach request
	// if !gossiper.OngoingFileSearch.IsOngoing {
	//    2.1) if search request has expired, do nothing/ return
	// return
	// }
	//    2.2) send the search reply to the SafeOngoingFileSearching chanel
	//        for handling (happens in the initiateFileSearching go routine)
	gossiper.OngoingFileSearch.SearchReplyChanel <- searchReply
}

func initiateFileSearching(gossiper *core.Gossiper, budget *uint64, searchKeywords *string) {
	// if budget is not specified, set it to 2 (default starting budget)
	fileSearch := gossiper.OngoingFileSearch
	searchReplyChanel := fileSearch.SearchReplyChanel
	searchBudget := *budget
	defaultBudget := false
	if searchBudget == 0 {
		defaultBudget = true
		searchBudget = uint64(2)
	}

	ticker := time.NewTicker(1 * time.Second)

	newSearchRequest := &core.SearchRequest{Origin: gossiper.Name, Budget: searchBudget, Keywords: strings.Split(*searchKeywords, ",")}
	HandlePeerSearchRequest(gossiper, newSearchRequest)

	for {
		select {
		case <-ticker.C:
			// every 1 second, repeat the query with double budget
			//  until reaching a maximum budget (32) or a threshold number of total
			//   matches (e.g. 2 for the tests)

			if defaultBudget {
				// check if budget exceeded maximum
				if searchBudget > constants.RingSearchBudgetLimit {
					//    if so, end the search, return
					// gossiper.OngoingFileSearch.IsOngoing = false
					return
				}

				//    otherwise, double the budget and send another request
				searchBudget *= 2
				newSearchRequest := &core.SearchRequest{Origin: gossiper.Name, Budget: searchBudget, Keywords: strings.Split(*searchKeywords, ",")}
				HandlePeerSearchRequest(gossiper, newSearchRequest)

				//      restart the ticker
				ticker = time.NewTicker(1 * time.Second)
			}

		case searchReply := <-searchReplyChanel:
			// we received a search reply, so let's process it
			// iterate over each of the received SearchResults
			searchResults := searchReply.Results
			currentMatches := gossiper.OngoingFileSearch.MatchesFound
			for _, res := range searchResults {
				helpers.PrintFileMatchFound(res.FileName, searchReply.Origin, hex.EncodeToString(res.MetafileHash), res.ChunkMap)
				//    for each, store information about the specified chunks
				if _, found := currentMatches[res.FileName]; found {
					// we have information about some chunks of this file
					infoSoFar := currentMatches[res.FileName]
					if infoSoFar.ChunkCount != uint64(len(infoSoFar.LocationOfChunks)) {
						// if we haven't found all the chunks of this file, then add more info
						for _, chunk := range res.ChunkMap {
							peersSoFar := make([]string, 0)
							if _, haveIt := infoSoFar.LocationOfChunks[chunk]; haveIt {
								// if we have info about this chunk, add the reply's origin
								peersSoFar = infoSoFar.LocationOfChunks[chunk]
							}
							peersSoFar = append(peersSoFar, searchReply.Origin)
							infoSoFar.LocationOfChunks[chunk] = peersSoFar
						}
					}
					// save updated information
					currentMatches[res.FileName] = infoSoFar
					gossiper.OngoingFileSearch.MatchesFound = currentMatches
				} else {
					// received search results for a new file (e.g. we don't have any info about it so far)
					newMatch := &core.FileSearchMatch{FileName: res.FileName, ChunkCount: res.ChunkCount, LocationOfChunks: make(map[uint64][]string), Metahash: res.MetafileHash}
					for _, ch := range res.ChunkMap {
						peersSoFar := newMatch.LocationOfChunks[ch]
						peersSoFar = append(peersSoFar, searchReply.Origin)
						newMatch.LocationOfChunks[ch] = peersSoFar
					}
					gossiper.OngoingFileSearch.MatchesFound[res.FileName] = newMatch
				}
				if gossiper.OngoingFileSearch.MatchesFound[res.FileName].ChunkCount ==
					uint64(len(gossiper.OngoingFileSearch.MatchesFound[res.FileName].LocationOfChunks)) {
					addNewMatchedFileName(gossiper, res.FileName)
				}
			}
			// at this point, check if we have reached two full matches
			fullMatchesCount := 0
			for _, searchMatch := range gossiper.OngoingFileSearch.MatchesFound {
				if searchMatch.ChunkCount == uint64(len(searchMatch.LocationOfChunks)) {
					fullMatchesCount++
				}
			}
			if fullMatchesCount >= constants.FullMatchesThreshold {
				//    if so, print "SEARCH FINISHED"
				helpers.PrintSearchFinished()
				return
			}
			ticker = time.NewTicker(1 * time.Second)

		}
	}

}

///////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////
//////////                   HELPERS                             //////////
///////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////

func addNewMatchedFileName(gossiper *core.Gossiper, fname string) {
	contains := false
	for _, fn := range gossiper.OngoingFileSearch.MatchesFileNames {
		if strings.Compare(fn, fname) == 0 {
			contains = true
		}
	}
	if !contains {
		gossiper.OngoingFileSearch.MatchesFileNames = append(gossiper.OngoingFileSearch.MatchesFileNames, fname)
	}
}

func forwardSearchReply(gossiper *core.Gossiper, msg *core.SearchReply) {
	if msg.HopLimit == 0 {
		// if we have reached the HopLimit, drop the message
		return
	}
	gossiper.DestinationTable.DsdvLock.Lock()
	forwardingAddress := gossiper.DestinationTable.Dsdv[msg.Destination]
	gossiper.DestinationTable.DsdvLock.Unlock()
	// If current node has no information about next hop to the destination in question
	if strings.Compare(forwardingAddress, "") == 0 {
		// fmt.Println("NO FORWARDING ADDRESS AGAIN :??")
		// TODO: What to do if there is no 'next hop' known when peer has to forward a private packet
	}

	// Decrement the HopLimit right before forwarding the packet
	msg.HopLimit--
	// Encode and send packet
	packetToSend := core.GossipPacket{SearchReply: msg}
	packetBytes, err := protobuf.Encode(&packetToSend)
	helpers.HandleErrorFatal(err)
	core.ConnectAndSend(forwardingAddress, gossiper.Conn, packetBytes)
}

func performLocalFilenameSearch(gossiper *core.Gossiper, keywords []string) []*core.SearchResult {
	knownFiles := gossiper.GetAllKnownFiles()
	searchResults := make([]*core.SearchResult, 0)

	for _, kw := range keywords {
		kw += ".*"
		for _, f := range knownFiles.MetaStringToFileInfo {
			fname := f.FileName
			matched, _ := regexp.MatchString(kw, fname)
			if matched {
				chunkMap := make([]uint64, 0)
				for idx, _ := range f.Metafile {
					chunkMap = append(chunkMap, uint64(idx))
				}
				newResult := &core.SearchResult{FileName: fname, MetafileHash: f.MetaHash[:], ChunkCount: f.ChunksCount, ChunkMap: chunkMap}
				searchResults = append(searchResults, newResult)
			}
		}
	}

	return searchResults
}
