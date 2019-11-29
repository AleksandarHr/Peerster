package core

import (
	"sync"
)

type FileSearchMatch struct {
	FileName   string
	ChunkCount uint64
	Metahash   []byte
	// maps chunk index to peer where we found the chunk at
	LocationOfChunks map[uint64]string
}

// A struct to hold information about the currently executed search request
type SafeOngoingFileSearching struct {
	SearchReplyChanel         chan *SearchReply
	SearchDownloadReplyChanel chan *DataReply
	IsOngoing                 bool
	Budget                    uint64
	Keywords                  *string
	DownloadedFiles           map[string]*FileInformation
	// maps from filename to FileSearchMatch struct
	MatchesFound      map[string]*FileSearchMatch
	SearchRequestLock sync.Mutex
}

func CreateSafeOngoingFileSearching(bdgt *uint64, keywords *string) *SafeOngoingFileSearching {
	searchChanel := make(chan *SearchReply)
	downloadChanel := make(chan *DataReply)
	matches := make(map[string]*FileSearchMatch)

	fileSearch := &SafeOngoingFileSearching{SearchReplyChanel: searchChanel, SearchDownloadReplyChanel: downloadChanel,
		Budget: *bdgt, Keywords: keywords, MatchesFound: matches, IsOngoing: true, DownloadedFiles: make(map[string]*FileInformation)}

	return fileSearch
}

func IsWholeFileFound(search *SafeOngoingFileSearching, fileName string) bool {
	fileFound := false
	if match, ok := search.MatchesFound[fileName]; ok {
		fileFound = (match.ChunkCount == uint64(len(match.LocationOfChunks)))
	}

	return fileFound
}
