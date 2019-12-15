package core

import (
	"sync"
)

type FileSearchMatch struct {
	FileName   string
	ChunkCount uint64
	Metahash   []byte
	// maps chunk index to peer where we found the chunk at
	LocationOfChunks map[uint64][]string
}

// A struct to hold information about the currently executed search request
type SafeOngoingFileSearching struct {
	SearchReplyChanel         chan *SearchReply
	SearchDownloadReplyChanel chan *DataReply
	IsOngoing                 bool
	// maps from filename to FileSearchMatch struct
	MatchesFound      map[string]*FileSearchMatch
	MatchesFileNames  []string
	SearchRequestLock sync.Mutex
}

func CreateSafeOngoingFileSearching() *SafeOngoingFileSearching {
	searchChanel := make(chan *SearchReply)
	downloadChanel := make(chan *DataReply)
	matches := make(map[string]*FileSearchMatch)
	names := make([]string, 0)

	fileSearch := &SafeOngoingFileSearching{SearchReplyChanel: searchChanel, SearchDownloadReplyChanel: downloadChanel,
		MatchesFound: matches, MatchesFileNames: names, IsOngoing: true}

	return fileSearch
}

func IsWholeFileFound(search *SafeOngoingFileSearching, fileName string) bool {
	fileFound := false
	if match, ok := search.MatchesFound[fileName]; ok {
		fileFound = (match.ChunkCount == uint64(len(match.LocationOfChunks)))
	}

	return fileFound
}
