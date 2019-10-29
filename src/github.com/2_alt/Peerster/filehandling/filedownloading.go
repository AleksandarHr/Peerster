package filehandling
import "github.com/2_alt/Peerster/core"
// import "crypto/sha256"
// import "os"
// import "fmt"
// import "io"
// import "bytes"
// import "path/filepath"
// import "encoding/hex"
// import "io/ioutil"

func createDataRequest(origin string, dest string, hopLimit uint32, hash []byte) *core.DataRequest{
  request := core.DataRequest{Origin: origin, Destination: dest, HopLimit: hopLimit, HashValue: hash}
  return &request
}

func downloadFile(gossiper *core.Gossiper, requestMetahash []byte, dest string) {
  defaultHopLimit := uint32(10)
  // Start download by first sending a DataRequest the metafile associated with 'metahash'
  dataReq := createDataRequest(gossiper.Name, dest, defaultHopLimit, requestMetahash)
  packetToSend := core.GossipPacket{DataRequest: dataReq}
  // TODO: Send packet to nextHop for the given destination name

  // Wait for a valid reply and make sure the hash of the reply mathes with the requested one

  // retransmit periodically the request if no DataReply arrives after a timeout(5 seconds)

  // Download each of the file chunks one at a time (keep a state of files being downloaded)
      // Save chunks for future sharing (only after completely downloaded)
      // Once all chunks are received, reconstruct the file and save it in _Downloads
}
