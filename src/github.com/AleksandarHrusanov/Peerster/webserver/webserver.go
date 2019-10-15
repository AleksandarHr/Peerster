package webserver

import "net/http"
import "net"
import "fmt"
import "encoding/json"
import "github.com/AleksandarHrusanov/Peerster/structs"
import "github.com/dedis/protobuf"

//MyPage - a struct
type MyPage struct {
  PeerID string
  KnownPeers []string
  SeenMessages []string
}

var myIndexPage *MyPage
var gossiperNode *structs.Gossiper
var receiveChanel = make(chan []byte)
var sendChanel = make(chan []byte)
// type OriginTextPair struct {
//   Origin string
//   Text string
// }

func latestRumomrMessageHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
  case "GET":
    // gets messages that were sent to the peer from toher peers and updates the chat box
    msgList := gossiperNode.GetLatestRumorMessagesList()
    msgListJSON, err := json.Marshal(msgList)
    fmt.Println("Length of list = ", len(msgList))
    if err != nil {
      fmt.Println("Error when serializing to json: ", err)
    }

    receiveChanel <- msgListJSON
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write(msgListJSON)
  case "POST":
    // gets messages that were sent to the peer from the client/GUI and updates the chat box
    r.ParseForm()
    fmt.Println(r.Form)
    msg := structs.Message{}
    msg.Text = r.FormValue("NewMsg")
    sendMessage := structs.OriginTextPair{Origin: gossiperNode.Name, Text: msg.Text}
    msgJSON, err := json.Marshal(sendMessage)
    if err != nil {
      fmt.Println("Error when serializing to json: ", err)
    }
    sendChanel <- msgJSON
  }
}


func peersHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
  case "GET":
    msgList := gossiperNode.Peers
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
  http.HandleFunc("/nodes", peersHandler)
  for {
    err := http.ListenAndServe(":3000", nil)
    if err !=nil {

    }
  }
}

/*HandleWebClientMessages - a function to listen for and handle web client messages */
func HandleWebClientMessages(UIPort string) {
  for {
    select {
    case newMsgReceived := <- receiveChanel:
      var originTxtPairs []structs.OriginTextPair
      err := json.Unmarshal(newMsgReceived, &originTxtPairs)
      if err != nil {
        fmt.Println("Error when deserializing from json: ", err)
      }
      fmt.Println("Number of paris received = ", len(originTxtPairs))
    case newMsgToSend := <- sendChanel:
      var originTxtPair structs.OriginTextPair
      err := json.Unmarshal(newMsgToSend, &originTxtPair)
      if err != nil {
        fmt.Println("Error when deserializing from json: ", err)
      }
      fmt.Println("origin = ", originTxtPair.Origin, " text = ", originTxtPair.Text)
      sendMsgToNode(UIPort, originTxtPair.Text)
    }
  }
}

func sendMsgToNode (UIPort string, msg string) {
  service := "127.0.0.1" + ":" + UIPort

  // Opens a UDP connection to send messages read-in from the console
  remoteAddress, err := net.ResolveUDPAddr("udp4", service)
  if err != nil {
    fmt.Println("Error resolving udp address")
    return
  }
  udpConn, err := net.DialUDP("udp", nil, remoteAddress)
  if err != nil {
    fmt.Println("Error dialing udp")
    return
  }

  defer udpConn.Close()

  msgStruct := &structs.Message{msg}
  packetBytes, err := protobuf.Encode(msgStruct)
  if err != nil {
    fmt.Println("Error encoding message, ", err)
  }

  _, err = udpConn.Write(packetBytes)
  if err != nil {
    fmt.Println("Error writing message, ", err)
    return
  }

}
