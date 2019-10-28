package helpers
import "crypto/sha256"
import "os"
import "fmt"
import "io"
// import "io/ioutil"
// import "bufio"

// fixed file chunk size of 8KB
const fixedChunkSize = uint32(8)
const filesFolder = "../_SharedFiles/"
const sha256HashSize = 32

func computeSha256(data []byte) [sha256HashSize]byte{
  return sha256.Sum256(data)
}

func scanFileInChunks(chunkSize uint32, fn string) [][]byte{
  // open the file
  filePath := filesFolder + fn
  file, err := os.Open(filePath)
  if err != nil {
    fmt.Println("Error opening a file: ", err)
    return nil
  }
  defer file.Close()

  fileChunks := make([][]byte, 0)
  buffer := make([]byte, chunkSize)
  // read file in chunks of 8kb 
  for {
    bytesRead, err := file.Read(buffer)
    if err!= nil {
      if err != io.EOF {
        fmt.Println("Error reading a file: ", err)
      }
      break
    }
    fileChunks = append(fileChunks, buffer[:bytesRead])
    fmt.Println("Number of bytes read: ", bytesRead)
    fmt.Println("bytestream to string: ", string(buffer[:bytesRead]))

  }
  return fileChunks
}

func hashAllFileChunks(chunks [][]byte, numberOfChunks uint32) []byte {
  // hashedChunks := make([]byte, sha256HashSize*numberOfChunks)
  // for _, chunk := range chunks {
    // currentHash := computeSha256(chunk)
    // hashedChunks = append(hashedChunks, currentHash)
  // }

  return nil;
}

func writeHashedChunksToFile(chunks []byte) *os.File{
  file, err := os.Create("metafile")
  defer file.Close()
  if err != nil {
    fmt.Println("Error creating binary metafile: ", err)
    return nil
  }

  _, err = file.Write(chunks)
  if err != nil {
    fmt.Println("Error writing to metafile: ", err)
    return nil
  }

  return file
}

func computeMetafileHash(file *os.File) {

}
