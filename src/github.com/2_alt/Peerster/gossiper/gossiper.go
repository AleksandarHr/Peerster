package gossiper

import (
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/2_alt/Peerster/core"
	"github.com/2_alt/Peerster/helpers"
)

// StartGossiper Start the gossiper
func StartGossiper(gossiperPtr *core.Gossiper, simplePtr *bool, antiEntropyPtr *int, routeRumorPtr *int) {
	rand.Seed(time.Now().UnixNano())
	cleanFileFoldersOnStartup("./_SharedFiles/chunks")
	cleanFileFoldersOnStartup("./_Downloads")
	cleanFileFoldersOnStartup("./_Downloads/chunks")
	// Listen from client and peers
	if !*simplePtr {
		go clientListener(gossiperPtr, *simplePtr)
		go peersListener(gossiperPtr, *simplePtr)
	} else {
		// In simple mode there is no anti-entropy so no infinite loop
		// to prevent the program to end
		go clientListener(gossiperPtr, *simplePtr)
		peersListener(gossiperPtr, *simplePtr)
	}
	defer gossiperPtr.Conn.Close()
	defer gossiperPtr.LocalConn.Close()

	// Send the initial route rumor message on startup
	go routeRumorHandler(gossiperPtr, routeRumorPtr)
	// go removeCompletedStates(gossiperPtr)
	// Anti-entropy
	if *antiEntropyPtr > 0 {
		for {
			time.Sleep(time.Duration(*antiEntropyPtr) * time.Second)

			if len(gossiperPtr.KnownPeers) > 0 {
				randomAddress := helpers.PickRandomInSlice(gossiperPtr.KnownPeers)
				sendStatus(gossiperPtr, randomAddress)
			}
		}
	} else {
		for {
			time.Sleep(time.Duration(*antiEntropyPtr) * 999)
		}
	}
}

func cleanFileFoldersOnStartup(folder string) error {

	if _, err := os.Stat(folder); os.IsNotExist(err) {
		os.Mkdir(folder, 0755)
	} else {
		dir, _ := filepath.Abs(folder)
		d, err := os.Open(dir)
		if err != nil {
			return err
		}
		defer d.Close()

		names, err := d.Readdirnames(-1)
		if err != nil {
			return err
		}
		for _, name := range names {
			err = os.RemoveAll(filepath.Join(dir, name))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// TODO: remove downloading states from memory when finished
// func removeCompletedStates(gossiper *core.Gossiper) {
// 	for {
// 		time.Sleep(5 * time.Second)
// 		gossiper.DownloadingLock.Lock()
// 		allStates := gossiper.DownloadingStates
// 		for downloadFrom, states := range allStates {
// 			for i, st := range states {
// 				if st.DownloadFinished {
// 					close(states[i].DownloadChanel)
// 					if len(states) == 1 {
// 						delete(gossiper.DownloadingStates, downloadFrom)
// 					} else {
// 						states[i] = states[len(states)-1]
// 						states = states[:len(states)-1]
// 					}
// 				}
// 			}
// 		}
// 		gossiper.DownloadingLock.Unlock()
// 	}
// }
