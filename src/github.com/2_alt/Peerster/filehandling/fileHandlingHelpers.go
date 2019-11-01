package filehandling

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/2_alt/Peerster/core"
	"github.com/2_alt/Peerster/helpers"
	"github.com/dedis/protobuf"
)

func createDataRequest(origin string, dest string, hash []byte) *core.DataRequest {
	request := core.DataRequest{Origin: origin, Destination: dest, HopLimit: uint32(10), HashValue: hash[:]}
	return &request
}

func createDataReply(origin string, dest string, hash []byte, data []byte) *core.DataReply {
	reply := core.DataReply{Origin: origin, Destination: dest, HopLimit: uint32(10), HashValue: hash[:], Data: data}
	return &reply
}

// TODO:  Duplicated code -REFACTOR!!!
// A function to forward a data request to the corresponding next hop
func forwardDataRequest(gossiper *core.Gossiper, msg *core.DataRequest) {

	if msg.HopLimit == 0 {
		// if we have reached the HopLimit, drop the message
		return
	}

	forwardingAddress := gossiper.DestinationTable[msg.Destination]
	// If current node has no information about next hop to the destination in question
	if strings.Compare(forwardingAddress, "") == 0 {
		fmt.Println("NO FORWARDING ADDRESS AGAIN :??")
		// TODO: What to do if there is no 'next hop' known when peer has to forward a private packet
	}

	// Decrement the HopLimit right before forwarding the packet
	msg.HopLimit--
	// Encode and send packet
	packetToSend := core.GossipPacket{DataRequest: msg}
	packetBytes, err := protobuf.Encode(&packetToSend)
	helpers.HandleErrorFatal(err)
	core.ConnectAndSend(forwardingAddress, gossiper.Conn, packetBytes)
}

// A function to forward a data request to the corresponding next hop
func forwardDataReply(gossiper *core.Gossiper, msg *core.DataReply) {
	if msg.HopLimit == 0 {
		// if we have reached the HopLimit, drop the message
		return
	}

	forwardingAddress := gossiper.DestinationTable[msg.Destination]
	// If current node has no information about next hop to the destination in question
	if strings.Compare(forwardingAddress, "") == 0 {
		// TODO: What to do if there is no 'next hop' known when peer has to forward a private packet
	}

	// Decrement the HopLimit right before forwarding the packet
	msg.HopLimit--
	// Encode and send packet
	packetToSend := core.GossipPacket{DataReply: msg}
	packetBytes, err := protobuf.Encode(&packetToSend)
	helpers.HandleErrorFatal(err)
	core.ConnectAndSend(forwardingAddress, gossiper.Conn, packetBytes)
}

// a function to resend the latest requested chunk
func resendDataRequest(gossiper *core.Gossiper, downloadFrom string) {
	if state, ok := gossiper.DownloadingStates[downloadFrom]; ok {
		chunkToRerequest := state.LatestRequestedChunk
		request := createDataRequest(gossiper.Name, downloadFrom, chunkToRerequest[:])
		forwardDataRequest(gossiper, request)
	}
}

// returns true if the received data reply corresponds to the latest requested hash
func replyWasExpected(requestHash []byte, reply *core.DataReply) bool {
	return bytes.Compare(requestHash, reply.HashValue) == 0
}

// returns true if the hash value in the reply corresponds to the data
func replyIntegrityCheck(reply *core.DataReply) bool {
	dataHash := computeSha256(reply.Data)
	actualHash := reply.HashValue
	return bytes.Compare(actualHash, dataHash[:]) == 0
}

func reconstructAndSaveFullyDownloadedFile(fileInfo *core.FileInformation) {
	// create a file
	fileData := make([]byte, 0)
	for _, chunk := range fileInfo.ChunksMap {
		// concatenate all file chunks into a single array
		fileData = append(fileData, chunk[:]...)
	}
	// create and write to file
	path, _ := filepath.Abs(downloadedFilesFolder)
	filePath, _ := filepath.Abs(path + "/" + fileInfo.FileName)
	ioutil.WriteFile(filePath, fileData[:], 0777)
}

func createDownloadingState(clientMsg *core.Message) *core.DownloadingState {
	downloadFrom := clientMsg.Destination
	fileName := clientMsg.File
	var requestedMetaHash [32]byte
	copy(requestedMetaHash[:], (*clientMsg.Request)[:32])
	fInfo := &core.FileInformation{FileName: *fileName, MetaHash: requestedMetaHash,
		Metafile: make(map[uint32][32]byte, 0), ChunksMap: make(map[string][8192]byte, 0)}
	state := core.DownloadingState{FileInfo: fInfo, DownloadFinished: false, MetafileDownloaded: false,
		MetafileRequested: true, NextChunkIndex: uint32(0), ChunksToRequest: make([][]byte, 0),
		DownloadingFrom: *downloadFrom, DownloadChanel: make(chan *core.DataReply)}
	return &state
}

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
