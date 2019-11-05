package filehandling

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"

	"github.com/2_alt/Peerster/constants"
)

// HandleFileIndexing - a function to index, divide, hash, and save hashed chunks of a file
func HandleFileIndexing(fname string) {

	filePath, _ := filepath.Abs(constants.SharedFilesFolder + fname)
	file, err := os.Open(filePath)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer file.Close()

	fInfo, _ := file.Stat()
	var fSize int64 = fInfo.Size()
	totalChunksCount := uint64(math.Ceil(float64(fSize) / float64(constants.FixedChunkSize)))
	// fmt.Println("Spliiting into ", totalChunksCount, " chunks.")

	metaFile := make(map[uint64][constants.HashSize]byte)

	if _, err := os.Stat(constants.ShareFilesChunksFolder); os.IsNotExist(err) {
		os.Mkdir(constants.ShareFilesChunksFolder, constants.FileMode)
	}

	for i := uint64(0); i < totalChunksCount; i++ {
		currChunkSize := int(math.Min(constants.FixedChunkSize,
			float64(fSize-int64(i*constants.FixedChunkSize))))
		// fmt.Println("Chunk ", i, " has size of ", currChunkSize, " bytes")
		buffer := make([]byte, currChunkSize)
		bytesRead, _ := file.Read(buffer)
		buffer = buffer[:bytesRead]
		hash := computeSha256(buffer)
		newName := hashToString(hash)
		chunkPath, _ := filepath.Abs(constants.ShareFilesChunksFolder + "/" + newName)
		_, err := os.Create(chunkPath)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		metaFile[i] = hash
		ioutil.WriteFile(chunkPath, buffer, os.ModeAppend)
	}

	appendedMetaFile := make([]byte, 0, constants.HashSize*len(metaFile))
	for i := uint64(0); i < totalChunksCount; i++ {
		hs := metaFile[i]
		appendedMetaFile = append(appendedMetaFile, hs[:]...)
	}

	metahash := computeSha256(appendedMetaFile)
	metahashString := hashToString(metahash)
	// fmt.Println("Metahash is === ", metahashString)
	metafilePath, _ := filepath.Abs(constants.ShareFilesChunksFolder + "/" + metahashString)
	ioutil.WriteFile(metafilePath, appendedMetaFile, constants.FileMode)
}
