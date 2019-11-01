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
const downloadedFilesFolder = "./_Downloads/"
const fixedChunkSize = 8192
const fileMode = 0755
const hashSize = 32

// HandleFileIndexing - a function to index, divide, hash, and save hashed chunks of a file
func HandleFileIndexing(fname string) {
	breakFileIntoChunks(fname)
}

func breakFileIntoChunks(fname string) {

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
	fmt.Println("Spliiting into ", totalChunksCount, " chunks.")

	metaFile := make(map[uint64][hashSize]byte)

	for i := uint64(0); i < totalChunksCount; i++ {
		currChunkSize := int(math.Min(fixedChunkSize, float64(fSize-int64(i*fixedChunkSize))))
		fmt.Println("Chunk ", i, " has size of ", currChunkSize, " bytes")
		buffer := make([]byte, currChunkSize)
		file.Read(buffer)

		hash := computeSha256(buffer)
		newName := hashToString(hash)
		chunkPath, _ := filepath.Abs(sharedFilesFolder + newName)
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
	metafilePath, _ := filepath.Abs(sharedFilesFolder + "/" + metahashString)
	fmt.Println("METAHASH is == ", metahashString, " With number of bytes == ", len(appendedMetaFile))
	ioutil.WriteFile(metafilePath, appendedMetaFile, fileMode)
}

// ========================================
//              Chunks Handling
// ========================================

// Handle requested hash from file system
// ======================================
func retrieveRequestedHashFromFileSystem(requestedHash [32]byte) []byte {
	hashBytes := getChunkOrMetafileFromFileSystem(requestedHash)
	if hashBytes != nil {
		// if the requested hash  was a filechunk
		return hashBytes
	}
	return nil
}

// if the given fileInfo has the requested chunk, return it
func getChunkOrMetafileFromFileSystem(chunkHash [32]byte) []byte {
	hashString := hashToString(chunkHash)
	// Look for chunk in the _SharedFiles folder
	sharedPath, _ := filepath.Abs(sharedFilesFolder + hashString)
	if _, err := os.Stat(sharedPath); err == nil {
		data, _ := ioutil.ReadFile(sharedPath)
		return data
	}

	// Look for chunk in the _DownloadedFiles folder
	downloadsPath, _ := filepath.Abs(downloadedFilesFolder + hashString)
	if _, err := os.Stat(downloadsPath); err == nil {
		data, _ := ioutil.ReadFile(downloadsPath)
		return data
	}

	return nil
}
