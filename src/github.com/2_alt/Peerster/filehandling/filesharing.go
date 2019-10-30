package filehandling
import "github.com/2_alt/Peerster/core"


// ShareFile - a function to share a file (metafile and chunks, one at a time)
//  e.g. when a client specifies the file to be indexed and sent to destination
func ShareFile(gossiper *core.Gossiper, fileInfo *core.FileInformation, dest string){
  // First, send metafile
  // reply := core.DataReply{Origin: gossiper.Name, Destination: dest, HopLimit: uint32(10),
  //                         HashValue: fileInfo.MetaHash, Data: fileInfo.Metafile}
  // packetToSend := core.GossipPacket{DataReply: &reply}
}
