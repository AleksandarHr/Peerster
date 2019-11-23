package core

import (
	"sync"
)

type FileSearchMatch struct {
	FileName   string
	ChunkCount uint64
	// maps chunk index to peer where we found the chunk at
	LocationOfChunks map[uint64]string
}

// A struct to hold information about the currently executed search request
type SafeOngoingFileSearching struct {
	SearchReplyChanel chan *SearchReply
	IsOngoing         bool
	Budget            uint64
	Keywords          []string
	// maps from filename to FileSearchMatch struct
	MatchesFound      map[string]*FileSearchMatch
	SearchRequestLock sync.Mutex
}

func CreateSafeOngoingFileSearching() *SafeOngoingFileSearching {
	ch := make(chan *SearchReply)
	matches := make(map[string]*FileSearchMatch)

	fileSearch := &SafeOngoingFileSearching{SearchReplyChanel: ch, MatchesFound: matches, IsOngoing: true}

	return fileSearch
}

func IsWholeFileFound(search *SafeOngoingFileSearching, fileName string) bool {
	fileFound := false
	if match, ok := search.MatchesFound[fileName]; ok {
		fileFound = (match.ChunkCount == uint64(len(match.LocationOfChunks)))
	}

	return fileFound
}
