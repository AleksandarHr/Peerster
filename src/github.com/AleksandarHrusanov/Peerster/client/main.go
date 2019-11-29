package main

import (
	"encoding/hex"
	"flag"
	"log"
	"os"
	"strings"

	"github.com/AleksandarHrusanov/Peerster/core"
)

func main() {
	// Parse the arguments
	uIPortPtr := flag.String("UIPort", "8080", "Port for the UI client (default \"8080\")")
	msgPtr := flag.String("msg", "", "message to be sent")
	destPtr := flag.String("dest", "", "message to be sent")
	fileToSharePtr := flag.String("file", "", "file to be indexed by the gossiper")
	requestHash := flag.String("request", "", "string representation of the metahash of the file to request")
	keywords := flag.String("keywords", "", "comma separated keywords used for file searching by name")
	budget := flag.Uint64("budget", uint64(2), "starting budget used for ring-expand search")
	flag.Parse()
	localAddressAndPort := "127.0.0.1:" + *uIPortPtr

	// Establish UDP connection and send the message
	checkFlags(uIPortPtr, msgPtr, destPtr, fileToSharePtr, requestHash, keywords)
	core.ClientConnectAndSend(localAddressAndPort, msgPtr, destPtr, fileToSharePtr, requestHash, keywords, budget)
}

func checkFlags(uiPortPtr, msgPtr, destPtr, fileToSharePtr, requestHash, keywordsPtr *string) {

	if strings.Compare(*uiPortPtr, "") == 0 {
		log.Fatal("No UIPort specified.")
		os.Exit(1)
	}

	if strings.Compare(*requestHash, "") != 0 {
		_, err := hex.DecodeString(*requestHash)
		if err != nil {
			log.Fatal("Unable to decode hash: ", err)
			os.Exit(1)
		}
	}

	// uiport + destination + file + request
	fileDownload := (strings.Compare(*destPtr, "") != 0 &&
		strings.Compare(*fileToSharePtr, "") != 0 &&
		strings.Compare(*requestHash, "") != 0 &&
		strings.Compare(*msgPtr, "") == 0)
	// uiport + file
	fileShare := (strings.Compare(*destPtr, "") == 0 &&
		strings.Compare(*fileToSharePtr, "") != 0 &&
		strings.Compare(*requestHash, "") == 0 &&
		strings.Compare(*msgPtr, "") == 0)

	// uiport + msg (+ destination)
	sendingMessage := (strings.Compare(*fileToSharePtr, "") == 0 &&
		strings.Compare(*requestHash, "") == 0 &&
		strings.Compare(*msgPtr, "") != 0)

	//
	fileSearch := (strings.Compare(*keywordsPtr, "") != 0 &&
		strings.Compare(*destPtr, "") == 0 &&
		strings.Compare(*msgPtr, "") == 0)

	if !(fileDownload || fileShare || sendingMessage || fileSearch) {
		log.Fatal("Combination of flags is not allowed.")
		os.Exit(1)
	}
}
