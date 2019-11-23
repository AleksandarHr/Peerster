package filesearching

import (
	"sync"

	"github.com/AleksandarHrusanov/Peerster/core"
)

type FileSearchMatch struct {
	FileName   string
	ChunkCount uint64
	// maps chunk index to peer where we found the chunk at
	LocationOfChunks map[uint64]string
}

// A struct to hold information about the currently executed search request
type SafeOngoingFileSearching struct {
	SearchReplyChanel chan *core.SearchReply
	IsOngoing         bool
	SearchRequestLock sync.Mutex
	Budget            uint64
	Keywords          []string
	// maps from filename to FileSearchMatch struct
	MatchesFound map[string]*FileSearchMatch
}

func CreateSafeOngoingFileSearching() *SafeOngoingFileSearching {
	ch := make(chan *core.SearchReply)
	matches := make(map[string]*FileSearchMatch)

	fileSearch := &SafeOngoingFileSearching{SearchReplyChanel: ch, MatchesFound: matches}

	return fileSearch
}

func IsWholeFileFound(search *SafeOngoingFileSearching, fileName string) bool {
	fileFound := false
	if match, ok := search.MatchesFound[fileName]; ok {
		fileFound = (match.ChunkCount == uint64(len(match.LocationOfChunks)))
	}

	return fileFound
}
