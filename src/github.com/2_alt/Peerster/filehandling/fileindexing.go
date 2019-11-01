package filehandling

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/2_alt/Peerster/core"
)

// fixed file chunk size of 8KB = fixedChunkSize bytes
const sharedFilesFolder = "./_SharedFiles/"
const downloadedFilesFolder = "./_Downloads/"
const fixedChunkSize = 8192
const fileMode = 0755
const hashSize = 32

// size of the sha-256 hash in bytes
// const core.Sha256HashSize = uint32(32)

// HandleFileIndexing - a function to index, divide, hash, and save hashed chunks of a file
func HandleFileIndexing(fname string) {
	fmt.Println("About to scanning, indexing and hashing")
	everythingAtOnce(fname)
	// fileInfo := scanIndexAndHashFile(fname)
	// fmt.Println("Done scanning, indexing and hashing")
	// saveHashesToFiles(fileInfo)
	// fmt.Println("Done saving hashes")
}

func everythingAtOnce(fn string) {

	chunksMap := make(map[string][fixedChunkSize]byte)
	metaFile := make(map[int][32]byte)
	chunkIndex := 0

	// read file in chunks of 8kb = fixedChunkSize bytes
	filePath, _ := filepath.Abs(sharedFilesFolder + fn)
	data, _ := ioutil.ReadFile(filePath)
	fileSize := len(data)

	for i := 0; i < fileSize; i += fixedChunkSize {
		bytesLeft := fileSize - i
		ch := make([]byte, fixedChunkSize)
		if bytesLeft >= fixedChunkSize {
			ch = data[i : i+fixedChunkSize]
		} else {
			ch = data[i:fileSize]
		}
		fmt.Println("Bytes read == ", len(ch), " to index == ", chunkIndex)
		var currChunk [fixedChunkSize]byte
		copy(currChunk[:], ch)
		hash := computeSha256(currChunk[:])
		chunksMap[hashToString(hash)] = currChunk
		metaFile[chunkIndex] = hash
		chunkIndex++

		chunkPath, _ := filepath.Abs(sharedFilesFolder + "/" + hashToString(hash))
		ioutil.WriteFile(chunkPath, currChunk[:], fileMode)
	}

	appendedMetaFile := make([]byte, 32*len(metaFile))
	for i := 0; i < len(metaFile); i++ {
		hs := metaFile[i]
		appendedMetaFile = append(appendedMetaFile, hs[:]...)
	}

	metahash := computeSha256(appendedMetaFile)
	metahashString := hashToString(metahash)
	metafilePath, _ := filepath.Abs(sharedFilesFolder + "/" + metahashString)
	fmt.Println("METAHASH is == ", metahashString, " With number of bytes == ", len(appendedMetaFile))
	ioutil.WriteFile(metafilePath, appendedMetaFile, fileMode)

}

func scanIndexAndHashFile(fn string) *core.FileInformation {

	// buffer := make([]byte, core.FixedChunkSize)
	chunksMap := make(map[string][fixedChunkSize]byte)
	metaFile := make(map[uint32][32]byte)
	chunkIndex := uint32(0)

	// read file in chunks of 8kb = fixedChunkSize bytes
	filePath, _ := filepath.Abs(sharedFilesFolder + fn)
	data, _ := ioutil.ReadFile(filePath)
	fileSize := len(data)

	for i := 0; i < fileSize; i += fixedChunkSize {
		bytesLeft := fileSize - i
		ch := make([]byte, fixedChunkSize)
		if bytesLeft >= fixedChunkSize {
			ch = data[i : i+fixedChunkSize]
		} else {
			ch = data[i:fileSize]
		}
		fmt.Println("Bytes read == ", len(ch), " to index == ", chunkIndex)
		var currChunk [fixedChunkSize]byte
		copy(currChunk[:], ch)
		hash := computeSha256(currChunk[:])
		chunksMap[hashToString(hash)] = currChunk
		metaFile[chunkIndex] = hash
		chunkIndex++
	}
	// for {
	// 	bytesRead, err := file.Read(buffer)
	// 	if err != nil {
	// 		if err != io.EOF {
	// 			fmt.Println("Error reading a file: ", err)
	// 			break
	// 		}
	// 	}
	// 	if bytesRead == 0 {
	// 		fmt.Println("No bytes read")
	// 		break
	// 	}
	// 	if bytesRead > fixedChunkSize {
	// 		fmt.Println("Chunk read was more than fixedChunkSize bytes")
	// 		break
	// 	}
	// to find the total number of bytes in the file
	// 	fileSize += bytesRead
	// 	var currChunk [fixedChunkSize]byte
	// 	copy(currChunk[:], buffer[:bytesRead])
	//
	// 	chunkHash := computeSha256(currChunk[:bytesRead])
	// 	hashString := hashToString(chunkHash)
	//
	// 	chunksMap[hashString] = currChunk
	// 	metaFile[chunkIndex] = chunkHash
	// 	chunkIndex++
	// }

	fileInfo := createFileInformation(fn, uint32(fileSize), metaFile, chunksMap)
	return fileInfo
}

func saveHashesToFiles(fileInfo *core.FileInformation) {
	metahashString := hashToString(fileInfo.MetaHash)
	bytesLeft := fileInfo.NumberOfBytes
	// uncomment and use instead if storing file chunks in file-specific folders
	// path, _ := filepath.Abs(sharedFilesFolder + "/" + metahashString)
	path, _ := filepath.Abs(sharedFilesFolder)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("Making a directory :: ", path)
		os.Mkdir(path, fileMode)
	} else {
		fmt.Println("Path exists: ", path)
	}

	chunksMap := fileInfo.ChunksMap
	for name, content := range chunksMap {
		chunkPath, _ := filepath.Abs(path + "/" + name)
		if bytesLeft < fixedChunkSize {
			ioutil.WriteFile(chunkPath, content[:bytesLeft], fileMode)
		} else {
			ioutil.WriteFile(chunkPath, content[:], fileMode)
		}
		bytesLeft -= fixedChunkSize
	}

	metafile := concatenateMetafile(fileInfo.Metafile)
	metafilePath, _ := filepath.Abs(path + "/" + metahashString)
	fmt.Println("METAHASH is == ", metahashString)
	ioutil.WriteFile(metafilePath, metafile, fileMode)
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
