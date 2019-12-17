package server

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AleksandarHrusanov/Peerster/core"
	"github.com/AleksandarHrusanov/Peerster/helpers"
	"github.com/gorilla/mux"
)

type handlerMaker struct {
	G *core.Gossiper
}

// Get the id
func (m *handlerMaker) idHandler(w http.ResponseWriter, r *http.Request) {
	goss := m.G

	switch r.Method {
	default:
		gossiperID := goss.Name
		gossiperIDJson, err := json.Marshal(gossiperID)
		helpers.HandleErrorFatal(err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(gossiperIDJson)
	}
}

// Serve index.html
func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "text/html")
	http.ServeFile(w, r, "./server/frontend/index.html")
}

// Handle message requests
func (m *handlerMaker) messageHandler(w http.ResponseWriter, r *http.Request) {
	goss := m.G

	switch r.Method {
	case http.MethodGet:
		// Return json of rumors
		msgList := goss.GetAllNonRouteRumors()
		msgListJSON, err := json.Marshal(msgList)
		helpers.HandleErrorFatal(err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(msgListJSON)

	case http.MethodPost:
		// Get the message
		reqBody, err := ioutil.ReadAll(r.Body)
		helpers.HandleErrorFatal(err)
		empty := ""
		zero := uint64(0)
		text := ""
		dest := ""
		fileToShare := ""
		hashRequest := ""
		err = json.Unmarshal(reqBody, &text)
		helpers.HandleErrorFatal(err)

		// Use the client to send the message to the gossiper
		core.ClientConnectAndSend(goss.GetLocalAddr(), &text, &dest, &fileToShare, &hashRequest, &empty, &zero)

		// Return json of rumors
		time.Sleep(50 * time.Millisecond)
		msgList := goss.GetAllRumors()
		msgListJSON, err := json.Marshal(msgList)
		helpers.HandleErrorFatal(err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(msgListJSON)
	}
}

func (m *handlerMaker) privateMessageHandler(w http.ResponseWriter, r *http.Request) {
	goss := m.G

	switch r.Method {
	case http.MethodGet:
		// Return json of private messages with the chosen origin
		msgList := goss.GetAllPrivateMessagesBetween()
		msgListJSON, err := json.Marshal(msgList)
		helpers.HandleErrorFatal(err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(msgListJSON)

	case http.MethodPost:
		// Get the message
		reqBody, err := ioutil.ReadAll(r.Body)
		helpers.HandleErrorFatal(err)
		var msg []string
		empty := ""
		zero := uint64(0)
		fileToShare := ""
		hashRequest := ""
		err = json.Unmarshal(reqBody, &msg)
		helpers.HandleErrorFatal(err)

		// Use the client to send the message to the gossiper
		if strings.Compare(msg[0], "") != 0 && strings.Compare(msg[1], "") != 0 {
			core.ClientConnectAndSend(goss.GetLocalAddr(), &msg[0], &msg[1], &fileToShare, &hashRequest, &empty, &zero)
		}

		// Return json of rumors
		time.Sleep(50 * time.Millisecond)
		msgList := goss.GetAllPrivateMessagesBetween()
		msgListJSON, err := json.Marshal(msgList)
		helpers.HandleErrorFatal(err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(msgListJSON)
	}
}

// Handle node requests
func (m *handlerMaker) originsHandler(w http.ResponseWriter, r *http.Request) {
	goss := m.G

	switch r.Method {
	case http.MethodGet:
		// Return json of knownpeers
		msgKnownOrigins := goss.GetAllKnownOrigins()

		if len(msgKnownOrigins) > 0 {
			msgKnownOriginsJSON, err := json.Marshal(msgKnownOrigins)
			helpers.HandleErrorFatal(err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(msgKnownOriginsJSON)
		}
	}
}

// Handle confirmed tlcs
func (m *handlerMaker) confirmedTLCsHandler(w http.ResponseWriter, r *http.Request) {
	goss := m.G

	switch r.Method {
	case http.MethodGet:
		confirmedTLCs := goss.GetConfirmedTLCs()
		tlcsToPrint := make([]string, 0)
		for _, tlc := range confirmedTLCs {
			origin := tlc.Origin
			name := tlc.TxBlock.Transaction.Name
			metahash := hex.EncodeToString(tlc.TxBlock.Transaction.MetafileHash)
			id := tlc.ID
			size := tlc.TxBlock.Transaction.Size
			toPrint := "CONFIRMED GOSSIP origin " + origin + " ID " + fmt.Sprint(id) + " file name " + name + " size " + strconv.FormatInt(size, 10) + " metahash " + metahash
			tlcsToPrint = append(tlcsToPrint, toPrint)
		}

		if len(tlcsToPrint) > 0 {
			tlcsToPrintJSON, err := json.Marshal(tlcsToPrint)
			helpers.HandleErrorFatal(err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(tlcsToPrintJSON)
		}
	}
}

// Handle node requests
func (m *handlerMaker) shareFilesHandler(w http.ResponseWriter, r *http.Request) {
	goss := m.G

	switch r.Method {
	case http.MethodGet:
		// Return json of knownpeers
		sharedFiles := goss.GetAllSharedFilesAndHashes()
		sharedFilesJSON, err := json.Marshal(sharedFiles)
		helpers.HandleErrorFatal(err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(sharedFilesJSON)

	case http.MethodPost:
		// Get the message
		reqBody, err := ioutil.ReadAll(r.Body)
		helpers.HandleErrorFatal(err)
		text := ""
		dest := ""
		empty := ""
		zero := uint64(0)
		fileToShare := ""
		hashRequest := ""
		err = json.Unmarshal(reqBody, &fileToShare)
		helpers.HandleErrorFatal(err)

		// Use the client to send the message to the gossiper
		core.ClientConnectAndSend(goss.GetLocalAddr(), &text, &dest, &fileToShare, &hashRequest, &empty, &zero)

		// Return json of rumors
		time.Sleep(50 * time.Millisecond)
		fileAndHash := fileToShare
		fileAndHashJSON, err := json.Marshal(fileAndHash)
		helpers.HandleErrorFatal(err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(fileAndHashJSON)
	}
}

// Handle node requests
func (m *handlerMaker) downloadFilesHandler(w http.ResponseWriter, r *http.Request) {
	goss := m.G

	switch r.Method {
	case http.MethodPost:
		reqBody, err := ioutil.ReadAll(r.Body)
		helpers.HandleErrorFatal(err)
		// -dest, -file, -request
		var msg []string
		txt := ""
		empty := ""
		zero := uint64(0)
		err = json.Unmarshal(reqBody, &msg)
		helpers.HandleErrorFatal(err)

		// Use the client to send the message to the gossiper
		if strings.Compare(msg[0], "") != 0 && strings.Compare(msg[1], "") != 0 {
			core.ClientConnectAndSend(goss.GetLocalAddr(), &txt, &msg[0], &msg[1], &msg[2], &empty, &zero)
		}
	}
}

// Handle implicit download
func (m *handlerMaker) implicitDownloadFilesHandler(w http.ResponseWriter, r *http.Request) {
	goss := m.G

	switch r.Method {
	case http.MethodPost:
		reqBody, err := ioutil.ReadAll(r.Body)
		helpers.HandleErrorFatal(err)
		// -dest, -file, -request
		var matchedFile string
		empty := ""
		zero := uint64(0)
		err = json.Unmarshal(reqBody, &matchedFile)
		helpers.HandleErrorFatal(err)
		hash := goss.GetMetafileHashByName(matchedFile)
		// Use the client to send the message to the gossiper
		if strings.Compare(matchedFile, "") != 0 && strings.Compare(hash, "") != 0 {
			core.ClientConnectAndSend(goss.GetLocalAddr(), &empty, &empty, &matchedFile, &hash, &empty, &zero)
		}
	}
}

// Handle node requests
func (m *handlerMaker) searchFile(w http.ResponseWriter, r *http.Request) {
	goss := m.G

	switch r.Method {
	case http.MethodPost:
		reqBody, err := ioutil.ReadAll(r.Body)
		helpers.HandleErrorFatal(err)
		// -dest, -file, -request
		empty := ""
		zero := uint64(0)
		keywords := ""
		err = json.Unmarshal(reqBody, &keywords)
		helpers.HandleErrorFatal(err)

		// Use the client to send the message to the gossiper
		if strings.Compare(keywords, "") != 0 {
			core.ClientConnectAndSend(goss.GetLocalAddr(), &empty, &empty, &empty, &empty, &keywords, &zero)
		}
	case http.MethodGet:
		// Return json of matchedFileNames
		matchedFiles := goss.GetAllFullyMatchedFilenames()

		if len(matchedFiles) > 0 {
			matchedFilesJSON, err := json.Marshal(matchedFiles)
			helpers.HandleErrorFatal(err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(matchedFilesJSON)
		}
	}
}

// Handle node requests
func (m *handlerMaker) nodeHandler(w http.ResponseWriter, r *http.Request) {
	goss := m.G

	switch r.Method {
	case http.MethodPost:
		reqBody, err := ioutil.ReadAll(r.Body)
		helpers.HandleErrorFatal(err)
		newAddress := ""
		err = json.Unmarshal(reqBody, &newAddress)
		helpers.HandleErrorFatal(err)

		// Add to gossiper knownpeers
		goss.AddPeer(newAddress)

		// Return json of knownpeers
		msgKnownPeers := goss.GetAllKnownPeers()
		msgKnownPeersJSON, err := json.Marshal(msgKnownPeers)
		helpers.HandleErrorFatal(err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(msgKnownPeersJSON)

	case http.MethodGet:
		// Return json of knownpeers
		msgKnownPeers := goss.GetAllKnownPeers()
		msgKnownPeersJSON, err := json.Marshal(msgKnownPeers)
		helpers.HandleErrorFatal(err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(msgKnownPeersJSON)
	}
}

// StartServer start the peer's server
func StartServer(g *core.Gossiper) {

	defaultServerPort := ":" + g.GetUIPort() // default ":8080"
	handlerMaker := &handlerMaker{g}

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", homeHandler)
	router.HandleFunc("/message", handlerMaker.messageHandler)
	router.HandleFunc("/id", handlerMaker.idHandler)
	router.HandleFunc("/node", handlerMaker.nodeHandler)
	router.HandleFunc("/origin", handlerMaker.originsHandler)
	router.HandleFunc("/share", handlerMaker.shareFilesHandler)
	router.HandleFunc("/private", handlerMaker.privateMessageHandler)
	router.HandleFunc("/download", handlerMaker.downloadFilesHandler)
	router.HandleFunc("/implicit_download", handlerMaker.implicitDownloadFilesHandler)
	router.HandleFunc("/search", handlerMaker.searchFile)
	router.HandleFunc("/confirmed_tlcs", handlerMaker.confirmedTLCsHandler)

	// Listen for http requests and serve them
	log.Fatal(http.ListenAndServe(defaultServerPort, router))
}
