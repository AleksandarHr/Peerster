package server

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"
	"github.com/2_alt/Peerster/gossiper"
	"github.com/2_alt/Peerster/helpers"
	"github.com/gorilla/mux"
)

type handlerMaker struct {
	G *gossiper.Gossiper
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
		msgList := goss.GetAllRumors()
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
		err = json.Unmarshal(reqBody, &text)
		helpers.HandleErrorFatal(err)

		// Use the client to send the message to the gossiper
		helpers.ClientConnectAndSend(goss.GetLocalAddr(), &text)

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
func StartServer(g *gossiper.Gossiper) {

	defaultServerPort := ":" + g.GetUIPort() // default ":8080"
	handlerMaker := &handlerMaker{g}

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", homeHandler)
	router.HandleFunc("/message", handlerMaker.messageHandler)
	router.HandleFunc("/id", handlerMaker.idHandler)
	router.HandleFunc("/node", handlerMaker.nodeHandler)

	// Listen for http requests and serve them
	log.Fatal(http.ListenAndServe(defaultServerPort, router))
}