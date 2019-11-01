package filehandling

import "os"
import "fmt"

import "path/filepath"

import "io/ioutil"
import "github.com/2_alt/Peerster/core"

// fixed file chunk size of 8KB = 8192 bytes
const sharedFilesFolder = "./_SharedFiles/"
const downloadedFilesFolder = "./_Downloads/"

// size of the sha-256 hash in bytes
// const core.Sha256HashSize = uint32(32)

// HandleFileIndexing - a function to index, divide, hash, and save hashed chunks of a file
func HandleFileIndexing(fname string) {
	fmt.Println("About to scanning, indexing and hashing")
	fileInfo := scanIndexAndHashFile(fname)
	fmt.Println("Done scanning, indexing and hashing")
	saveHashesToFiles(fileInfo)
	fmt.Println("Done saving hashes")
}

func scanIndexAndHashFile(fn string) *core.FileInformation {
	// open the file
	filePath, _ := filepath.Abs(sharedFilesFolder + fn)
	// file, err := os.Open(filePath)
	// if err != nil {
	// 	fmt.Println("Error opening a file: ", err)
	// }
	// defer file.Close()

	// buffer := make([]byte, core.FixedChunkSize)
	chunksMap := make(map[string][8192]byte)
	metaFile := make(map[uint32][32]byte)
	chunkIndex := uint32(0)

	// read file in chunks of 8kb = 8192 bytes
	data, _ := ioutil.ReadFile(filePath)
	fileSize := len(data)

	for i := 0; i < fileSize; i += 8192 {
		bytesLeft := fileSize - i
		ch := make([]byte, 8192)
		if bytesLeft >= 8192 {
			ch = data[i : i+8192]
		} else {
			ch = data[i:fileSize]
		}
		fmt.Println("Bytes read == ", len(ch), " to index == ", chunkIndex)
		hash := computeSha256(ch)
		var currChunk [8192]byte
		copy(currChunk[:], ch)
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
	// 	if bytesRead > 8192 {
	// 		fmt.Println("Chunk read was more than 8192 bytes")
	// 		break
	// 	}
	// to find the total number of bytes in the file
	// 	fileSize += bytesRead
	// 	var currChunk [8192]byte
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
		os.Mkdir(path, 0755)
	} else {
		fmt.Println("Path exists: ", path)
	}

	chunksMap := fileInfo.ChunksMap
	for name, content := range chunksMap {
		chunkPath, _ := filepath.Abs(path + "/" + name)
		if bytesLeft < 8192 {
			ioutil.WriteFile(chunkPath, content[:bytesLeft], 0755)
		} else {
			ioutil.WriteFile(chunkPath, content[:], 0755)
		}
		bytesLeft -= 8192
	}

	metafile := newConcatenateMetafile(fileInfo.ChunksMap)
	metafilePath, _ := filepath.Abs(path + "/" + metahashString)
	fmt.Println("METAHASH is == ", metahashString)
	ioutil.WriteFile(metafilePath, metafile, 0755)
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
