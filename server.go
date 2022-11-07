package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"os"
	"os/signal"
	"time"
)

var done chan interface{}
var interrupt chan os.Signal

// This does almost nothing; simply spits out the message just received (including responses to keepalive messages)
func receiveHandler(connection *websocket.Conn) {
	defer close(done)
	//store raw messages from last N minutes in msgArr
	var msgArr []ReceivedRisMessage
	for {
		_, msg, err := connection.ReadMessage()
		if err != nil {
			log.Println("Error in receive:", err)
			return
		}
		var data ReceivedRisMessage
		//unmarshal msg from byte array into data structure
		json.Unmarshal(msg, &data)
		msgArr = append(msgArr, data)

		//loop through msgArr, every msg w/o timestamp in range is removed
		for i := 0; i < len(msgArr); i++ {
			now := time.Now()
			ts := now.Unix()
			//currently stores within last minute --> need to decide N minutes
			if dataTs := msgArr[i].Data.Timestamp; dataTs < float64(ts)-60 {
				msgArr = append(msgArr[:i], msgArr[i+1:]...)
			}
		}

		// map to store the peer_asn as key with array of paths as the value
		// need to discuss if this is best practice --> will this make it easy to create graph data to pass to frontend?
		pathMap := make(map[string][][]int64)

		//need to count for multiple paths being presented from ris live
		//have array of arrays, accounting for multiple paths

		// need to discuss --> will this be maintained? what is the deciding factor for removing data from this map?
		pathMap[data.Data.PeerAsn] = append(pathMap[data.Data.PeerAsn], data.Data.Path)

		log.Println(pathMap)
		log.Printf("Received: %s\n", msg)
	}
}

type RisMessageData struct {
	Host   string `json:"host,omitempty"`
	Prefix string `json:"prefix,omitempty"`
}

type RisMessage struct {
	Type string          `json:"type"`
	Data *RisMessageData `json:"data"`
}

// ReceivedRisMessageData This is only a struct of required types from the ris-live documentation
// Should decide what optional fields might be important to look at
type ReceivedRisMessageData struct {
	Timestamp float64 `json:"timestamp"`
	Peer      string  `json:"peer"`
	PeerAsn   string  `json:"peer_asn"`
	ID        string  `json:"id"`
	Host      string  `json:"host"`
	Type      string  `json:"type"`
	Path      []int64 `json:"path"`
}

type ReceivedRisMessage struct {
	Type string                  `json:"type"`
	Data *ReceivedRisMessageData `json:"data"`
}

func main() {
	log.Println("hi")

	done = make(chan interface{})    // Channel to indicate that the receiverHandler is done
	interrupt = make(chan os.Signal) // Channel to listen for interrupt signal to terminate gracefully

	signal.Notify(interrupt, os.Interrupt) // Notify the interrupt channel for SIGINT

	socketUrl := "ws://ris-live.ripe.net/v1/ws/"
	conn, _, err := websocket.DefaultDialer.Dial(socketUrl, nil)
	if err != nil {
		log.Fatal("Error connecting to Websocket Server:", err)
	}
	defer conn.Close()
	go receiveHandler(conn)

	/* Ping message (re-used every minute or so */
	//ping := RisMessage{"ping", nil}
	//pingstr, err := json.Marshal(ping)
	//if err != nil {
	//	log.Fatal("Error marshalling ping message (!)")
	//}

	/* Subscribe */
	//subscription1 := RisMessage{"ris_subscribe", &RisMessageData{"rrc21", "151.101.0.0/22"}}
	subscription1 := RisMessage{"ris_subscribe", &RisMessageData{"", "151.101.0.0/22"}}
	if err != nil {
		log.Fatal("Error marshalling subscription message (!)")
	}
	log.Println("Subscribing to: ", subscription1)
	out1, err := json.Marshal(subscription1)
	conn.WriteMessage(websocket.TextMessage, out1)

	//subscription2 := RisMessage{"ris_subscribe", &RisMessageData{"rrc21", "2a04:4e42::/48"}}
	subscription2 := RisMessage{"ris_subscribe", &RisMessageData{"", "2a04:4e42::/48"}}
	if err != nil {
		log.Fatal("Error marshalling subscription message (!)")
	}
	log.Println("Subscribing to: ", subscription2)
	out2, err := json.Marshal(subscription2)
	conn.WriteMessage(websocket.TextMessage, out2)

	for {
		select {
		//case <-time.After(time.Duration(60) * time.Millisecond * 1000):
		//	// Send an echo packet 60 seconds
		//	err := conn.WriteMessage(websocket.TextMessage, pingstr)
		//	if err != nil {
		//		log.Println("Error during writing to websocket:", err)
		//		return
		//	}

		case <-interrupt:
			// We received a SIGINT; clean up
			log.Println("Received SIGINT interrupt signal. Closing all pending connections")
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("Error during closing websocket: ", err)
				return
			}

			select {
			case <-done:
				log.Println("Receiver channel closed, exiting")
			case <-time.After(time.Duration(1) * time.Second):
				log.Println("Timeout in closing receiving channel; exiting")
			}
			return
		}
	}
}
