package main

import (
	"flag"
	"github.com/2_alt/Peerster/core"
)

func main() {
	// Parse the arguments
	UIPortPtr := flag.String("UIPort", "8080", "Port for the UI client (default \"8080\")")
	msgPtr := flag.String("msg", "", "message to be sent")
	destPtr := flag.String("dest", "", "message to be sent")
	fileToSharePtr := flag.String("file", "", "file to be indexed by the gossiper")
	flag.Parse()
	localAddressAndPort := "127.0.0.1:" + *UIPortPtr

	// Establish UDP connection and send the message
	core.ClientConnectAndSend(localAddressAndPort, msgPtr, destPtr, fileToSharePtr)
}
