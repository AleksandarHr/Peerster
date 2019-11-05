package filehandling

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
)

// fixed file chunk size of 8KB = fixedChunkSize bytes
const sharedFilesFolder = "./_SharedFiles/"
const shareFilesChunksFolder = "./_SharedFiles/chunks"
const downloadedFilesFolder = "./_Downloads/"
const downloadedFilesChunksFolder = "./_Downloads/chunks"
const fixedChunkSize = 8192
const fileMode = 0755
const hashSize = 32

// HandleFileIndexing - a function to index, divide, hash, and save hashed chunks of a file
func HandleFileIndexing(fname string) {

	filePath, _ := filepath.Abs(sharedFilesFolder + fname)
	file, err := os.Open(filePath)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer file.Close()

	fInfo, _ := file.Stat()
	var fSize int64 = fInfo.Size()
	totalChunksCount := uint64(math.Ceil(float64(fSize) / float64(fixedChunkSize)))
	// fmt.Println("Spliiting into ", totalChunksCount, " chunks.")

	metaFile := make(map[uint64][hashSize]byte)

	if _, err := os.Stat(shareFilesChunksFolder); os.IsNotExist(err) {
		os.Mkdir(shareFilesChunksFolder, fileMode)
	}

	for i := uint64(0); i < totalChunksCount; i++ {
		currChunkSize := int(math.Min(fixedChunkSize, float64(fSize-int64(i*fixedChunkSize))))
		// fmt.Println("Chunk ", i, " has size of ", currChunkSize, " bytes")
		buffer := make([]byte, currChunkSize)
		bytesRead, _ := file.Read(buffer)
		buffer = buffer[:bytesRead]
		hash := computeSha256(buffer)
		newName := hashToString(hash)
		chunkPath, _ := filepath.Abs(shareFilesChunksFolder + "/" + newName)
		_, err := os.Create(chunkPath)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		metaFile[i] = hash
		ioutil.WriteFile(chunkPath, buffer, os.ModeAppend)
	}

	appendedMetaFile := make([]byte, 0, hashSize*len(metaFile))
	for i := uint64(0); i < totalChunksCount; i++ {
		hs := metaFile[i]
		appendedMetaFile = append(appendedMetaFile, hs[:]...)
	}

	metahash := computeSha256(appendedMetaFile)
	metahashString := hashToString(metahash)
	// fmt.Println("Metahash is === ", metahashString)
	metafilePath, _ := filepath.Abs(shareFilesChunksFolder + "/" + metahashString)
	ioutil.WriteFile(metafilePath, appendedMetaFile, fileMode)
}
