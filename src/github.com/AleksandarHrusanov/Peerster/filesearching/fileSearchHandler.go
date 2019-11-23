package filesearching

import (
	"strings"
	"time"

	"github.com/AleksandarHrusanov/Peerster/core"
	"github.com/AleksandarHrusanov/Peerster/helpers"
	"github.com/dedis/protobuf"
)

// A function to handle a search request coming from the client of this peerster node
func HandleClientSearchRequest(gossiper *core.Gossiper, clientSearchRequest *core.Message) {
	// start a new search request - declare and initialize an SafeOngoingFileSearching variable
	fileSearch := core.CreateSafeOngoingFileSearching()

	// save it in the gossipere
	gossiper.OngoingFileSearch = fileSearch
	// fire a new thread to handle this
	go initiateFileSearching(gossiper)
}

// A function to handle a search request coming from another peerster node
func HandlePeerSearchRequest(gossiper *core.Gossiper, searchRequest *core.SearchRequest) {
	// 1) Detect and discard duplicate requests (e.g. same Origin and same Keywords) in the last 0.5 seconds
	searchSignature := append([]string{searchRequest.Origin}, searchRequest.Keywords...)
	duplicateSignature := strings.Join(searchSignature, ",")

	//  store somewhere/somehow recent search requests - check if contains this
	gossiper.RecentSearches.SearchesLock.Lock()
	if _, searched := gossiper.RecentSearches.Searches[duplicateSignature]; searched {
		//      if so, return
		gossiper.RecentSearches.SearchesLock.Unlock()
		return
	}
	//      otherwise, update recent search requests and TIME 0.5 seconds somehow (go routine which
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
	keywordsMatches := make([]*core.SearchResult, 0) // TODO:: populate this []*SearchResult array
	//   If any matches found, create and send a Search Reply (using next hop?)
	if len(keywordsMatches) != 0 {
		searchReply := &core.SearchReply{Origin: gossiper.Name, Destination: searchRequest.Origin, HopLimit: uint32(10), Results: keywordsMatches}
		forwardSearchReply(gossiper, searchReply)
	}

	// 4) subtract 1 from the request's budget
	searchRequest.Budget--
	if searchRequest.Budget <= 0 {
		//    4.2) If remaining budget is <= 0, do nothing, return
		return
	}
	//    4.1) If remaining budget is greater than 0, redistribute the remainig
	//         budget as evenly as possible to up to B neighboring nodes
	//         every peer gets a search request with budget = integer part of (budget / #neighbours)
	//         and then iteratively add 1 to the budget of the first R neighbors (where R = budget % # neighbors)
	gossiper.PeersLock.Lock()
	peers := gossiper.KnownPeers
	baseBudget := searchRequest.Budget / len(peers)
	extraBudget := searchRequest.Budget % len(peers)
	idx := 0
	for ; idx < extraBudget; idx++ {
		tempBudget := baseBudget + 1
		// SEND basebudget + 1 to extraBudget number of peers
		newSearchRequest := &core.SearchRequest{Origin: searchRequest.Origin, Budget: uint64(tempBudget), Keywords: searchRequest.Keywords}
		packetToSend := &core.GossipPacket{SearchRequest: newSearchRequest}
		packetBytes, err := protobuf.Encode(&packetToSend)
		helpers.HandleErrorFatal(err)
		core.ConnectAndSend(peers[idx], gossiper.Conn, packetBytes)
	}
	for ; idx < len(peers); idx++ {
		// send basebudget to the rest of the peers
		newSearchRequest := &core.SearchRequest{Origin: searchRequest.Origin, Budget: uint64(baseBudget), Keywords: searchRequest.Keywords}
		packetToSend := &core.GossipPacket{SearchRequest: newSearchRequest}
		packetBytes, err := protobuf.Encode(&packetToSend)
		helpers.HandleErrorFatal(err)
		core.ConnectAndSend(peers[idx], gossiper.Conn, packetBytes)
	}
	gossiper.PeersLock.Unlock()
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
	if !gossiper.OngoingFileSearch.IsOngoing {
		//    2.1) if search request has expired, do nothing/ return
		return
	}
	//    2.2) send the search reply to the SafeOngoingFileSearching chanel
	//        for handling (happens in the initiateFileSearching go routine)
	gossiper.OngoingFileSearch.SearchReplyChanel <- searchReply
}

func initiateFileSearching(gossiper *core.Gossiper) {

	// if budget is not specified, set it to 2 (default starting budget)

	searchReplyChanel := gossiper.OngoingFileSearch.SearchReplyChanel
	ticker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-ticker.C:
			// every 1 second, repeat the query with double budget
			//  until reaching a maximum budget (32) or a threshold number of total
			//   matches (e.g. 2 for the tests)

			// check if budget exceeded maximum
			//    if so, end the search, return
			//    otherwise, double the budget and send another request
			//      restart the ticker

		case searchReply := <-searchReplyChanel:
			// we received a search reply, so let's process it
			// iterate over each of the received SearchResults
			//    for each, store information about the specified chunks
			// at this point, check if we have reached two full matches
			//    if so, print "SEARCH FINISHED"
			//    issue a download for the fully matched files
			//      NOTE: do not specify destination - instead, use the internally
			//      saved information about which node has which chunks
			//      From the metahash in the search result, reconstruct the metafile bytes
			//      and read the corresponding 32-bit regions, encode them to strings and issue
			//      separate 'chunk download requests to peers'.
		}
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
	packetToSend := &core.GossipPacket{SearchReply: msg}
	packetBytes, err := protobuf.Encode(&packetToSend)
	helpers.HandleErrorFatal(err)
	core.ConnectAndSend(forwardingAddress, gossiper.Conn, packetBytes)
}
