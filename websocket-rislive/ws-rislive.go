package main

import (
	"encoding/json"
	"log"
	"net/url"

	rislive "github.com/a16/go-rislive/pkg/message"
	"github.com/gorilla/websocket"
)

func main() {
	values := url.Values{}
	values.Add("client", "go-rislive-gorilla")
	u := url.URL{
		Scheme:   "wss",
		Host:     "ris-live.ripe.net:443",
		Path:     "/v1/ws/",
		RawQuery: values.Encode(),
	}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer func(c *websocket.Conn) {
		err := c.Close()
		if err != nil {

		}
	}(c)

	if err := c.WriteJSON(rislive.NewRisRequestRrcList()); err != nil {
		log.Println("write:", err)
		return
	}

	_, message, err := c.ReadMessage()
	if err != nil {
		log.Println("read:", err)
		return
	}

	var msg rislive.RisLiveMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Println("unmarshal:", err)
		return
	}
	if msg.Type != "ris_rrc_list" {
		log.Println("Received unexpected message: ", msg.Type)
		return
	}
	log.Println(msg.Data.(rislive.RisRrcList))
}
