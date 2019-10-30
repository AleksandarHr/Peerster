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
const downloadedFilesFolder = "./_Downloaded/"
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
  hashesMap := make(map[string][]byte)
  chunksMap := make(map[string][]byte)
  // read file in chunks of 8kb = 8192 bytes
  for {
    bytesRead, err := file.Read(buffer)
    if err!= nil {
      if err != io.EOF {
        fmt.Println("Error reading a file: ", err)
        break
      }
    }
    if bytesRead == 0 {
      fmt.Println("No bytes read")
      break;
    }
    fmt.Println("Number of bytes read == ", bytesRead)
    // to find the total number of bytes in the file
    fileSize += bytesRead
    currChunk := buffer[:bytesRead]

    chunkHash := computeSha256(currChunk)
    hashString := hashToString(chunkHash)

    chunksMap[hashString] = currChunk
    hashesMap[hashString] = chunkHash
  }

  fileInfo := createFileInformation(fn, uint32(fileSize), hashesMap, chunksMap)
  return fileInfo
}

func saveHashesToFiles(fileInfo *core.FileInformation) {

  metahashString := hashToString(fileInfo.MetaHash)
  // uncomment and use instead if storing file chunks in file-specific folders
  // path, _ := filepath.Abs(sharedFilesFolder + "/" + metahashString)
  path, _ := filepath.Abs(sharedFilesFolder)
  if _, err := os.Stat(path); os.IsNotExist(err) {
    fmt.Println("Making a directory :: ", path)
    os.Mkdir(path, 0777)
  } else {
    fmt.Println("Path exists: ", path)
  }

  chunksMap := fileInfo.HashedChunksMap
  for name, content := range chunksMap {
    chunkPath, _ := filepath.Abs(path + "/" + name)
    ioutil.WriteFile(chunkPath, content[:], 0777)
  }

  metafile := fileInfo.Metafile
  metafilePath, _ := filepath.Abs(path + "/" + metahashString)
  ioutil.WriteFile(metafilePath, metafile, 0777)
}

// ========================================
//              Chunks Handling
// ========================================

// Handle requested hash from file system
// ======================================
func retrieveRequestedHashFromFileSystem(requestedHash []byte) []byte {
  hashBytes := getChunkOrMetafileFromFileSystem(requestedHash)
  if hashBytes != nil {
    // if the requested hash  was a filechunk
    return hashBytes
  }
  return nil
}

// if the given fileInfo has the requested chunk, return it
func getChunkOrMetafileFromFileSystem (chunkHash []byte) []byte {
  hashString := hashToString(chunkHash)
  path, _ := filepath.Abs(sharedFilesFolder + hashString)
  if _, err := os.Stat(path); err == nil {
    data, _ := ioutil.ReadFile(path)
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
func getChunkFromStruct (fileInfo *core.FileInformation, chunkHash []byte) []byte {
  hashString := hashToString(chunkHash)
  if chunk, ok := fileInfo.HashedChunksMap[hashString]; ok {
    return chunk[:]
  }
  return nil
}

// If the metahash requested is of the current metafile, return the metafile
func getMetafileFromStruct (fileInfo *core.FileInformation, metahash []byte) []byte {
  if bytes.Compare(fileInfo.MetaHash[:], metahash[:]) == 0 {
    return fileInfo.Metafile
  }
  return nil
}


// ========================================
//              HELPERS
// ========================================
func createFileInformation (name string, numBytes uint32, hashedChunks map[string][]byte,
                            dataChunks map[string][]byte) *core.FileInformation {
  fileInfo := core.FileInformation{FileName: name}
  fileInfo.Metafile = concatenateHashedChunks(hashedChunks)
  fileInfo.NumberOfBytes = uint32(numBytes)
  fileInfo.HashedChunksMap = hashedChunks
  fileInfo.MetaHash = computeSha256(fileInfo.Metafile)
  fileInfo.ChunksMap = dataChunks
  return &fileInfo
}

func computeSha256(data []byte) []byte{
  hash := sha256.Sum256(data)
  return hash[:]
}

func concatenateHashedChunks(chunks map[string][]byte) []byte {
  metafile := make([]byte, 0)
  for _, ch := range chunks {
    metafile = append(metafile, ch[:]...)
  }

  return metafile
}

func hashToString(hash []byte) string {
  return hex.EncodeToString(hash[:])
}
