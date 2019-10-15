package webserver

import "net/http"
import "fmt"
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

func latestRumomrMessageHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
  case "GET":
    msgList := gossiperNode.GetLatestRumorMessagesList()
    msgListJSON, err := json.Marshal(msgList)
    if err != nil {
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write(msgListJSON)
  case "POST":
    r.ParseForm()
    fmt.Println(r.Form)
    msg := structs.Message{}
    msg.Text = r.FormValue("NewMsg")
    fmt.Println(msg.Text)
  }
}

func peersHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
  case "GET":
    msgList := gossiperNode.GetLatestRumorMessagesList()
    msgListJSON, err := json.Marshal(msgList)
    if err != nil {
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write(msgListJSON)
  case "POST":

  }
}

/*StartWebServer - a function */
func StartWebServer(gossiper *structs.Gossiper) {
  gossiperNode = gossiper
  // r := mux.NewRouter()
  http.Handle("/", http.FileServer(http.Dir("./")))
  // r.HandleFunc("/messages", latestRumomrMessageHandler)
  http.HandleFunc("/messages", latestRumomrMessageHandler)
  for {
    err := http.ListenAndServe(":3000", nil)
    if err !=nil {

    }
  }
}
