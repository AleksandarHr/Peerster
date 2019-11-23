package filesearching

import "github.com/AleksandarHrusanov/Peerster/core"

// A function to handle a search request coming from the client of this peerster node
func HandleClientSearchRequest(gossiper *core.Gossiper, clientSearchRequest *core.Message) {
	// start a new search request - declare and initialize an SafeOngoingFileSearching variable
	// save it in the gossiper
	// go initiateFileSearching() - fire a new thread to handle this
}

// A function to handle a search request coming from another peerster node
func HandlePeerSearchRequest(gossiper *core.Gossiper, searchRequest *core.SearchRequest) {
	// 1) Detect and discard duplicate requests (e.g. same Origin and same Keywords) in the last 0.5 seconds
	//  store somewhere/somehow recent search requests - check if contains this
	//      if so, return
	//      otherwise, update recent search requests and TIME 0.5 seconds somehow (go routine which
	//      deletes the message from the recent search requests after 0.5 seconds?)

	// 2) process the search request locally (and possibly send a SearchReply)
	//   Check both _SharedFiles and _Downloades (gossiper memory) for keywords matches
	//   If any matches found, create and send a Search Reply (using next hop?)

	// 4) subtract 1 from the request's budget
	//    4.2) If remaining budget is <= 0, do nothing, return
	//    4.1) If remaining budget is greater than 0, redistribute the remainig
	//         budget as evenly as possible to up to B neighboring nodes
	//         every peer gets a search request with budget = integer part of (budget / #neighbours)
	//         and then iteratively add 1 to the budget of the first R neighbors (where R = budget % # neighbors)
}

// A function to handle a search reply coming from another peerster node
func HandlePeerSearchReply(gossiper *core.Gossiper, searchReply *core.SearchReply) {
	// NOTE: assume all search replies correspond to previously-issued search requests
	// 1) if destination field is not the current gossiper's name, forward with hop limit
	// 2) if current node was the destination of the serach request
	//    2.1) if search request has expired, do nothing/ return
	//    2.2) send the search reply to the SafeOngoingFileSearching chanel
	//        for handling (happens in the initiateFileSearching go routine)
}

func initiateFileSearching(gossiper *core.Gossiper) {

  // if budget is not specified, set it to 2 (default starting budget)

  searchReplyChanel := gossiper.OngoingFileSearch.ch
  ticker := time.NewTicker(1 time.Second)

  for {
    select {
    case <- ticker.C:
      // every 1 second, repeat the query with double budget
      //  until reaching a maximum budget (32) or a threshold number of total
      //   matches (e.g. 2 for the tests)

      // check if budget exceeded maximum
      //    if so, end the search, return
      //    otherwise, double the budget and send another request
      //      restart the ticker

    case searchReply <- searchReplyChanel:
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
