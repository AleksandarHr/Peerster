package filehandling
import "github.com/2_alt/Peerster/core"
// import "crypto/sha256"
// import "os"
import "fmt"
// import "io"
// import "bytes"
// import "path/filepath"
// import "encoding/hex"
// import "io/ioutil"

func downloadFile(gossiper *core.Gossiper, fname string, requestMetahash [core.Sha256HashSize]byte, dest string) {
  // defaultHopLimit := uint32(10)
  // fileInfo := core.FileInformation{FileName: fname, MetaHash: requestMetahash}
  // // Create a DownloadingState and update gossiper's struct to keep track of what's going on
  // state := core.DownloadingState {FileInfo: &fileInfo, DownloadFinished: false,
  //                                 NextChunkIndex: uint32(0), DownloadingFrom: dest}
  // hashStr := hashToString(requestMetahash)
  // if otherDownloads, ok := gossiper.DownloadingStates[hashStr]; ok {
  //   // if we are already downloading other files from the same peer, add the new state
  //   otherDownloads = append(otherDownloads, &state)
  //   gossiper.DownloadingStates[hashStr] = otherDownloads
  // } else {
  //   // if this is the first file we are downloading from this peer
  //   downloads := make([]*core.DownloadingState, 0)
  //   downloads = append(downloads, &state)
  //   gossiper.DownloadingStates[hashStr] = downloads
  // }
  //
  //
  // // Start download by first sending a DataRequest the metafile associated with 'metahash'
  // dataReq := createDataRequest(gossiper.Name, dest, defaultHopLimit, requestMetahash)
  // packetToSend := core.GossipPacket{DataRequest: dataReq}
  // fmt.Println(packetToSend.DataRequest.Origin)
  // TODO: Send packet to nextHop for the given destination name

  // Wait for a valid reply and make sure the hash of the reply mathes with the requested one

  // retransmit periodically the request if no DataReply arrives after a timeout(5 seconds)

  // Download each of the file chunks one at a time (keep a state of files being downloaded)
      // Save chunks for future sharing (only after completely downloaded)
      // Once all chunks are received, reconstruct the file and save it in _Downloads
}
