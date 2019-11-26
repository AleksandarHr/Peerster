package constants

// SharedFilesFolder - a relative path for the _SharedFiles from the main Peerster executable
const SharedFilesFolder = "./_SharedFiles/"

// ShareFilesChunksFolder - a relative path for the _SharedFiles/chunks from the main Peerster executable
const ShareFilesChunksFolder = "./_SharedFiles/chunks"

// DownloadedFilesFolder - a relative path for the _Downloads from the main Peerster executable
const DownloadedFilesFolder = "./_Downloads/"

// DownloadedFilesChunksFolder - a relative path for the _Downloads/chunks from the main Peerster executable
const DownloadedFilesChunksFolder = "./_Downloads/chunks"

// FixedChunkSize = the chunk limit of 8KB
const FixedChunkSize = 8192

// FileMode - mode for creating files/directories
const FileMode = 0755

// HashSize = the size of a sha256 hash
const HashSize = 32

const StartingRingSearchBudget = uint64(2)

const RingSearchBudgetLimit = uint64(32)

const FullMatchesThreshold = 2
