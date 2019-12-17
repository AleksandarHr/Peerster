package filehandling

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/AleksandarHrusanov/Peerster/constants"
	"github.com/AleksandarHrusanov/Peerster/core"
	"github.com/AleksandarHrusanov/Peerster/helpers"
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
	gossiper.DestinationTable.DsdvLock.Lock()
	forwardingAddress := gossiper.DestinationTable.Dsdv[msg.Destination]
	gossiper.DestinationTable.DsdvLock.Unlock()
	// If current node has no information about next hop to the destination in question
	if strings.Compare(forwardingAddress, "") == 0 {
		// fmt.Println("NO FORWARDING ADDRESS AGAIN :??")
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
	gossiper.DestinationTable.DsdvLock.Lock()
	forwardingAddress := gossiper.DestinationTable.Dsdv[msg.Destination]
	gossiper.DestinationTable.DsdvLock.Unlock()
	// If current node has no information about next hop to the destination in question
	if strings.Compare(forwardingAddress, "") == 0 {
		// fmt.Println(" NO ADDRESS????????????????????????")
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
func resendDataRequest(gossiper *core.Gossiper, downloadFrom string, chunkToRerequest [constants.HashSize]byte) {
	if _, ok := gossiper.DownloadingStates[downloadFrom]; ok {
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
	dataHash := computeSha256(reply.Data[:len(reply.Data)])
	actualHash := reply.HashValue
	return bytes.Compare(actualHash, dataHash[:]) == 0
}

func reconstructAndSaveFullyDownloadedFile(fileInfo *core.FileInformation) {
	// create a file
	fileData := make([]byte, 0)
	numChunks := len(fileInfo.Metafile)
	for i := uint32(1); i <= uint32(numChunks); i++ {
		chunkHash := fileInfo.Metafile[i]
		chunk := fileInfo.ChunksMap[hashToString(chunkHash)]
		fileData = append(fileData, chunk[:]...)
	}
	// create and write to file
	path, _ := filepath.Abs(constants.DownloadedFilesFolder)
	filePath, _ := filepath.Abs(path + "/" + fileInfo.FileName)
	ioutil.WriteFile(filePath, fileData[:], 0777)
}

func createDownloadingState(clientMsg *core.Message) *core.DownloadingState {
	downloadFrom := clientMsg.Destination
	hashValue := convertSliceTo32Fixed(*clientMsg.Request)
	fileName := clientMsg.File

	var requestedMetaHash [constants.HashSize]byte
	copy(requestedMetaHash[:], (*clientMsg.Request)[:constants.HashSize])
	fInfo := &core.FileInformation{FileName: *fileName, MetaHash: requestedMetaHash,
		Metafile: make(map[uint32][32]byte, 0), ChunksMap: make(map[string][]byte, 0)}
	state := core.DownloadingState{FileInfo: fInfo, DownloadFinished: false, MetafileDownloaded: false,
		MetafileRequested: true, NextChunkIndex: uint32(0), ChunksToRequest: make([][]byte, 0),
		DownloadingFrom: *downloadFrom, DownloadChanel: make(chan *core.DataReply), LatestRequestedChunk: hashValue}
	return &state
}

func createFileInformation(name string, numBytes uint32, metafile map[uint32][constants.HashSize]byte,
	dataChunks map[string][]byte) *core.FileInformation {
	fileInfo := core.FileInformation{FileName: name}
	// fileInfo.Metafile = concatenateMetafile(hashedChunks)
	fileInfo.Metafile = metafile
	fileInfo.MetaHash = computeSha256(concatenateMetafile(metafile))
	fileInfo.ChunksMap = dataChunks
	return &fileInfo
}

func computeSha256(data []byte) [constants.HashSize]byte {
	hash := sha256.Sum256(data)
	return hash
}

func concatenateMetafile(chunks map[uint32][constants.HashSize]byte) []byte {
	metafile := make([]byte, 0)

	for i := 0; i < len(chunks); i++ {
		ch := chunks[uint32(i)]
		metafile = append(metafile, ch[:]...)
	}
	return metafile
}

func mapifyMetafile(mfile []byte) map[uint32][constants.HashSize]byte {
	metafile := make(map[uint32][constants.HashSize]byte, 0)

	for i := 0; i < len(mfile); i += constants.HashSize {
		if len(mfile) < i+constants.HashSize {
			bytes := convertSliceTo32Fixed(mfile[i:len(mfile)])
			metafile[uint32(i/constants.HashSize)+1] = bytes
		} else {
			bytes := convertSliceTo32Fixed(mfile[i : i+constants.HashSize])
			metafile[uint32(i/constants.HashSize)+1] = bytes
		}
	}
	return metafile
}

func hashToString(hash [constants.HashSize]byte) string {
	return hex.EncodeToString(hash[:])
}

func convertSliceTo32Fixed(slice []byte) [constants.HashSize]byte {
	var result [constants.HashSize]byte
	if len(slice) < constants.HashSize {
		copy(result[:], slice[:len(slice)])
	} else {
		copy(result[:], slice[:constants.HashSize])
	}
	return result
}

// ========================================
//              Chunks Handling
// ========================================

// Handle requested hash from file system
// ======================================
func retrieveRequestedHashFromFileSystem(requestedHash [constants.HashSize]byte) []byte {
	hashBytes := getChunkOrMetafileFromFileSystem(requestedHash)
	if hashBytes != nil {
		// if the requested hash  was a filechunk
		return hashBytes
	}
	return nil
}

func retrieveRequestedHashFromGossiperMemory(gossiper *core.Gossiper, requestedHash [constants.HashSize]byte) []byte {
	allChunks := gossiper.FilesAndMetahashes.AllChunks
	metahashes := gossiper.FilesAndMetahashes.MetaHashes
	hashString := hashToString(requestedHash)
	if hashBytes, ok := metahashes[hashString]; ok {
		return hashBytes[:]
	}

	return allChunks[hashString]
}

// if the given fileInfo has the requested chunk, return it
func getChunkOrMetafileFromFileSystem(chunkHash [constants.HashSize]byte) []byte {
	// Look for chunk in the _SharedFiles folder
	sharedPath := buildChunkPath(constants.ShareFilesChunksFolder, chunkHash[:])
	if _, err := os.Stat(sharedPath); err == nil {
		data, _ := ioutil.ReadFile(sharedPath)
		return data
	}

	// Look for chunk in the _DownloadedFiles folder
	// downloadsPath, _ := filepath.Abs(constants.DownloadedFilesChunksFolder + "/" + hashString)
	downloadsPath := buildChunkPath(constants.DownloadedFilesChunksFolder, chunkHash[:])
	if _, err := os.Stat(downloadsPath); err == nil {
		data, _ := ioutil.ReadFile(downloadsPath)
		return data
	}

	return nil
}

func wasLastFileChunk(gossiper *core.Gossiper, reply *core.DataReply, state *core.DownloadingState) bool {
	metafile := state.FileInfo.Metafile
	lastChunkInMetafile := state.FileInfo.Metafile[uint32(len(metafile)-1)]
	return bytes.Compare(reply.HashValue, lastChunkInMetafile[:]) == 0
}

func buildChunkPath(folder string, hashValue []byte) string {
	chunkPath, _ := filepath.Abs(folder + "/" + hashToString(convertSliceTo32Fixed(hashValue)))
	return chunkPath
}

func handleReceivedMetafile(gossiper *core.Gossiper, reply *core.DataReply, fname string, state *core.DownloadingState) {
	// read the metafile and populate hashedChunks in the file
	metafile := mapifyMetafile(reply.Data)
	gossiper.DownloadingLock.Lock()
	state.FileInfo.Metafile = metafile
	state.MetafileDownloaded = true
	gossiper.DownloadingLock.Unlock()

	// write metafile to file system
	// metafilePath := buildChunkPath(constants.DownloadedFilesChunksFolder, reply.HashValue)
	// ioutil.WriteFile(metafilePath, reply.Data, constants.FileMode)
}
