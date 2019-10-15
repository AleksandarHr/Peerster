package webserver

import "net/http"
import "github.com/AleksandarHrusanov/Peerster/structs"

//MyPage - a struct
type MyPage struct {
  PeerID string
  KnownPeers []string
  SeenMessages []string
}

var myIndexPage *MyPage

func getLatestRumomrMessageHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
  case "GET":
    // msgList := gossiper.GetLatestRumorMessagesList()
    // msgListJSON, err := json.Marhsal(list)
    // if err != nil {
    // }
    // w.Header().Set("Content-Type", "application/json")
    // w.WriteHeader(http.StatusOK)
    // w.Write(testJSON)
  }
}

/*StartWebServer - a function */
func StartWebServer(gossiper *structs.Gossiper) {
  http.Handle("/", http.FileServer(http.Dir("./")))
  http.HandleFunc("/messages", getLatestRumomrMessageHandler)
  for {
    err := http.ListenAndServe(":3000", nil)
    if err !=nil {

    }
  }
}
