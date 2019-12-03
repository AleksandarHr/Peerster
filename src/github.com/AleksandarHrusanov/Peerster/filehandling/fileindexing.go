package filehandling

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/AleksandarHrusanov/Peerster/constants"
	"github.com/AleksandarHrusanov/Peerster/core"
)

// HandleFileIndexing - a function to index, divide, hash, and save hashed chunks of a file
func HandleFileIndexing(gossiper *core.Gossiper, fname string) *core.FileInformation {

	filePath, _ := filepath.Abs(constants.SharedFilesFolder + fname)
	file, err := os.Open(filePath)
	fileSize := int64(0)

	fileInfo := &core.FileInformation{FileName: fname, Metafile: make(map[uint32][constants.HashSize]byte),
		ChunksMap: make(map[string][]byte)}

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer file.Close()

	fInfo, _ := file.Stat()
	var fSize int64 = fInfo.Size()
	totalChunksCount := uint64(math.Ceil(float64(fSize) / float64(constants.FixedChunkSize)))
	fileInfo.ChunksCount = totalChunksCount

	metaFile := make(map[uint32][constants.HashSize]byte)

	if _, err := os.Stat(constants.ShareFilesChunksFolder); os.IsNotExist(err) {
		os.Mkdir(constants.ShareFilesChunksFolder, constants.FileMode)
	}

	for i := uint64(0); i < totalChunksCount; i++ {
		currChunkSize := int(math.Min(constants.FixedChunkSize,
			float64(fSize-int64(i*constants.FixedChunkSize))))
		buffer := make([]byte, currChunkSize)
		bytesRead, _ := file.Read(buffer)
		fileSize += int64(bytesRead)
		buffer = buffer[:bytesRead]
		hash := computeSha256(buffer)
		newName := hashToString(hash)

		metaFile[uint32(i+1)] = hash
		fileInfo.ChunksMap[newName] = buffer
		gossiper.FilesAndMetahashes.FilesLock.Lock()
		gossiper.FilesAndMetahashes.AllChunks[newName] = buffer
		gossiper.FilesAndMetahashes.FilesLock.Unlock()
	}

	fileInfo.Metafile = metaFile
	appendedMetaFile := make([]byte, 0, constants.HashSize*len(metaFile))
	for i := uint64(0); i < totalChunksCount; i++ {
		hs := metaFile[uint32(i+1)]
		appendedMetaFile = append(appendedMetaFile, hs[:]...)
	}

	metahash := computeSha256(appendedMetaFile)
	metahashString := hashToString(metahash)
	fileInfo.MetaHash = metahash
	fileInfo.Size = fileSize
	gossiper.FilesAndMetahashes.FilesLock.Lock()
	gossiper.FilesAndMetahashes.FileNamesToMetahashesMap[fname] = metahashString
	gossiper.FilesAndMetahashes.MetaStringToFileInfo[metahashString] = fileInfo
	gossiper.FilesAndMetahashes.MetaHashes[metahashString] = appendedMetaFile
	gossiper.FilesAndMetahashes.FilesLock.Unlock()

	return fileInfo
}
