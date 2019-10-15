package webserver

import "net/http"
import "encoding/json"
import "github.com/AleksandarHrusanov/Peerster/structs"

//MyPage - a struct
type MyPage struct {
  PeerID string
  KnownPeers []string
  SeenMessages []string
}

var myIndexPage *MyPage
var gossiperNode *structs.Gossiper

func getLatestRumomrMessageHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
  case "GET":
    msgList := gossiperNode.GetLatestRumorMessagesList()
    msgListJSON, err := json.Marshal(msgList)
    if err != nil {
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write(msgListJSON)
  }
}

/*StartWebServer - a function */
func StartWebServer(gossiper *structs.Gossiper) {
  gossiperNode = gossiper
  http.Handle("/", http.FileServer(http.Dir("./")))
  http.HandleFunc("/messages", getLatestRumomrMessageHandler)
  for {
    err := http.ListenAndServe(":3000", nil)
    if err !=nil {

    }
  }
}
