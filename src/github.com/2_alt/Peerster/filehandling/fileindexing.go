package filehandling

import "os"
import "fmt"
import "io"
import "bytes"
import "path/filepath"
import "crypto/sha256"
import "encoding/hex"
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
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening a file: ", err)
	}
	defer file.Close()

	fileSize := 0

	buffer := make([]byte, core.FixedChunkSize)
	chunksMap := make(map[string][8192]byte)
	metaFile := make(map[uint32][32]byte)
	chunkIndex := uint32(0)
	// read file in chunks of 8kb = 8192 bytes
	for {
		bytesRead, err := file.Read(buffer)
		if err != nil {
			if err != io.EOF {
				fmt.Println("Error reading a file: ", err)
				break
			}
		}
		if bytesRead == 0 {
			fmt.Println("No bytes read")
			break
		}
		if bytesRead > 8192 {
			fmt.Println("Chunk read was more than 8192 bytes")
			break
		}
		// fmt.Println("Number of bytes read == ", bytesRead)
		// to find the total number of bytes in the file
		fileSize += bytesRead
		var currChunk [8192]byte
		copy(currChunk[:], buffer[:bytesRead])

		chunkHash := computeSha256(currChunk[:bytesRead])
		hashString := hashToString(chunkHash)

		chunksMap[hashString] = currChunk
		metaFile[chunkIndex] = chunkHash
		chunkIndex++
	}

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
		os.Mkdir(path, 0777)
	} else {
		fmt.Println("Path exists: ", path)
	}

	chunksMap := fileInfo.ChunksMap
	for name, content := range chunksMap {
		chunkPath, _ := filepath.Abs(path + "/" + name)
		if bytesLeft < 8192 {
			ioutil.WriteFile(chunkPath, content[:bytesLeft], 0777)
		} else {
			ioutil.WriteFile(chunkPath, content[:], 0777)
		}
		bytesLeft -= 8192
	}

	metafile := concatenateMetafile(fileInfo.Metafile)
	metafilePath, _ := filepath.Abs(path + "/" + metahashString)
	fmt.Println("METAHASH is == ", metahashString)
	ioutil.WriteFile(metafilePath, metafile, 0777)
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

// Handle requested hash from fileInformation struct
//==================================================
func handleRequestedHashFromStruct(fileInfo *core.FileInformation, requestedHash []byte) []byte {
	fileChunk := getChunkFromStruct(fileInfo, requestedHash)
	if fileChunk != nil {
		// if the requested hash  was a filechunk
		return fileChunk
	}

	metafile := getMetafileFromStruct(fileInfo, requestedHash)
	if metafile != nil {
		// if the requested hash was the metafile
		return metafile
	}

	return nil
}

// if the given fileInfo has the requested chunk, return it
func getChunkFromStruct(fileInfo *core.FileInformation, chunkHash []byte) []byte {
	hashString := hashToString(convertSliceTo32Fixed(chunkHash))
	if chunk, ok := fileInfo.ChunksMap[hashString]; ok {
		return chunk[:]
	}
	return nil
}

// If the metahash requested is of the current metafile, return the metafile
func getMetafileFromStruct(fileInfo *core.FileInformation, metahash []byte) []byte {
	if bytes.Compare(fileInfo.MetaHash[:], metahash[:]) == 0 {
		return concatenateMetafile(fileInfo.Metafile)
	}
	return nil
}

// ========================================
//              HELPERS
// ========================================
func createFileInformation(name string, numBytes uint32, metafile map[uint32][32]byte,
	dataChunks map[string][8192]byte) *core.FileInformation {
	fileInfo := core.FileInformation{FileName: name}
	// fileInfo.Metafile = concatenateMetafile(hashedChunks)
	fileInfo.Metafile = metafile
	fileInfo.NumberOfBytes = uint32(numBytes)
	fileInfo.MetaHash = computeSha256(concatenateMetafile(metafile))
	fileInfo.ChunksMap = dataChunks
	return &fileInfo
}

func computeSha256(data []byte) [32]byte {
	hash := sha256.Sum256(data[:len(data)])
	return hash
}

func concatenateMetafile(chunks map[uint32][32]byte) []byte {
	metafile := make([]byte, 0)

	for i := 0; i < len(chunks); i++ {
		ch := chunks[uint32(i)]
		fmt.Println("Appending a chunk with ID = ", i, "and length = ", len(ch), " and a hash = ", hashToString(ch))
		metafile = append(metafile, ch[:]...)
	}
	return metafile
}

func mapifyMetafile(mfile []byte) map[uint32][32]byte {
	metafile := make(map[uint32][32]byte, 0)

	for i := 0; i < len(mfile); i += 32 {
		if len(mfile) <= i+32 {
			metafile[uint32(i/32)] = convertSliceTo32Fixed(mfile[i:len(mfile)])
		} else {
			metafile[uint32(i/32)] = convertSliceTo32Fixed(mfile[i : i+32])
		}
	}
	return metafile
}

func hashToString(hash [32]byte) string {
	return hex.EncodeToString(hash[:])
}
