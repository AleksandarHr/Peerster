package server

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
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
		text := ""
		dest := ""
		fileToShare := ""
		hashRequest := ""
		err = json.Unmarshal(reqBody, &text)
		helpers.HandleErrorFatal(err)

		// Use the client to send the message to the gossiper
		core.ClientConnectAndSend(goss.GetLocalAddr(), &text, &dest, &fileToShare, &hashRequest)

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
		text := ""
		// dest := ""
		// fileToShare := ""
		// hashRequest := ""
		err = json.Unmarshal(reqBody, &text)
		helpers.HandleErrorFatal(err)

		// Use the client to send the message to the gossiper
		// core.ClientConnectAndSend(goss.GetLocalAddr(), &text, &dest, &fileToShare, &hashRequest)

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

// Handle node requests
func (m *handlerMaker) originsHandler(w http.ResponseWriter, r *http.Request) {
	goss := m.G

	switch r.Method {
	case http.MethodGet:
		// Return json of knownpeers
		msgKnownOrigins := goss.GetAllKnownOrigins()
		msgKnownOriginsJSON, err := json.Marshal(msgKnownOrigins)
		helpers.HandleErrorFatal(err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(msgKnownOriginsJSON)
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
		fileToShare := ""
		hashRequest := ""
		err = json.Unmarshal(reqBody, &fileToShare)
		helpers.HandleErrorFatal(err)

		// Use the client to send the message to the gossiper
		core.ClientConnectAndSend(goss.GetLocalAddr(), &text, &dest, &fileToShare, &hashRequest)

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

	// Listen for http requests and serve them
	log.Fatal(http.ListenAndServe(defaultServerPort, router))
}
