package filehandling
import "crypto/sha256"
import "os"
import "fmt"
import "io"
import "bytes"
import "path/filepath"
import "encoding/hex"
import "io/ioutil"
import "github.com/2_alt/Peerster/core"

// fixed file chunk size of 8KB = 8192 bytes
const fixedChunkSize = uint32(8192)
const filesFolder = "./_SharedFiles/"
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
  filePath, _ := filepath.Abs(filesFolder + fn)
  file, err := os.Open(filePath)
  if err != nil {
    fmt.Println("Error opening a file: ", err)
  }
  defer file.Close()

  fileSize := 0

  buffer := make([]byte, fixedChunkSize)
  chunksMap := make(map[string][core.Sha256HashSize]byte)
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

    chunksMap[hashString] = chunkHash
  }

  fileInfo := createFileInformation(fn, uint32(fileSize), chunksMap)
  return fileInfo
}

func saveHashesToFiles(fileInfo *core.FileInformation) {

  metahashString := hashToString(fileInfo.MetaHash)
  // uncomment and use instead if storing file chunks in file-specific folders
  // path, _ := filepath.Abs(filesFolder + "/" + metahashString)
  path, _ := filepath.Abs(filesFolder)
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
func handleRequestedHashFromFileSystem(fileInfo *core.FileInformation, requestedHash [core.Sha256HashSize]byte) []byte {
  hashBytes := getChunkOrMetafileFromFileSystem(fileInfo, requestedHash)
  if hashBytes != nil {
    // if the requested hash  was a filechunk
    return hashBytes
  }
  return nil
}

// if the given fileInfo has the requested chunk, return it
func getChunkOrMetafileFromFileSystem (fileInfo *core.FileInformation, chunkHash [core.Sha256HashSize]byte) []byte {
  hashString := hashToString(chunkHash)
  path, _ := filepath.Abs(filesFolder + hashString)
  if _, err := os.Stat(path); err == nil {
    data, _ := ioutil.ReadFile(path)
    return data
  }
  return nil
}

// Handle requested hash from fileInformation struct
//==================================================
func handleRequestedHashFromStruct(fileInfo *core.FileInformation, requestedHash [core.Sha256HashSize]byte) []byte {
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
func getChunkFromStruct (fileInfo *core.FileInformation, chunkHash [core.Sha256HashSize]byte) []byte {
  hashString := hashToString(chunkHash)
  if chunk, ok := fileInfo.HashedChunksMap[hashString]; ok {
    return chunk[:]
  }
  return nil
}

// If the metahash requested is of the current metafile, return the metafile
func getMetafileFromStruct (fileInfo *core.FileInformation, metahash [core.Sha256HashSize]byte) []byte {
  if bytes.Compare(fileInfo.MetaHash[:], metahash[:]) == 0 {
    return fileInfo.Metafile
  }
  return nil
}


// ========================================
//              HELPERS
// ========================================
func createFileInformation (name string, numBytes uint32, hashedChunks map[string][core.Sha256HashSize]byte) *core.FileInformation {
  fileInfo := core.FileInformation{FileName: name}
  fileInfo.Metafile = concatenateHashedChunks(hashedChunks)
  fileInfo.NumberOfBytes = uint32(numBytes)
  fileInfo.HashedChunksMap = hashedChunks
  fileInfo.MetaHash = computeSha256(fileInfo.Metafile)
  return &fileInfo
}

func computeSha256(data []byte) [core.Sha256HashSize]byte{
  return sha256.Sum256(data)
}

func concatenateHashedChunks(chunks map[string][core.Sha256HashSize]byte) []byte {
  metafile := make([]byte, 0)
  for _, ch := range chunks {
    metafile = append(metafile, ch[:]...)
  }

  return metafile
}

func hashToString(hash [32]byte) string {
  return hex.EncodeToString(hash[:])
}
