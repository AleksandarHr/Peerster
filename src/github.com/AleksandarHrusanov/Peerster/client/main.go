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
	keywords := flag.String("keywords", "", "string of comma-separated keywords used for file search")
	budget := flag.Int("budget", 2, "initial budget used for the expanding ring search")
	flag.Parse()
	localAddressAndPort := "127.0.0.1:" + *uIPortPtr

	// Establish UDP connection and send the message
	checkFlags(uIPortPtr, msgPtr, destPtr, fileToSharePtr, requestHash)
	core.ClientConnectAndSend(localAddressAndPort, msgPtr, destPtr, fileToSharePtr, requestHash, keywords, budget)
}

func checkFlags(uiPortPtr, msgPtr, destPtr, fileToSharePtr, requestHash *string) {

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

	if !(fileDownload || fileShare || sendingMessage) {
		log.Fatal("Combination of flags is not allowed.")
		os.Exit(1)
	}
}
