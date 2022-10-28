package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	_ "log"
	"net/http"
	_ "net/http"
)

// struct to hold info for ws connection
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Home Page")
}

func wsEndpoint(w http.ResponseWriter, r *http.Request) {

	// determines if incoming request from a diff domain is allowed to connect
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	// upgrade connection to websocket connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	// connected!
	log.Println("Client Connected")

	// send msgs from go app
	// once client connected, does this
	err = ws.WriteMessage(1, []byte("hi Client!"))
	if err != nil {
		log.Println(err)
	}

	// listen indefinitely for new msgs
	reader(ws)
}

// defines reader to listen for msgs being sent to ws endpoint
// takes in pointer to ws connection
func reader(conn *websocket.Conn) {
	for {
		// read in msg
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		// print msg
		fmt.Println(string(p))

		if err := conn.WriteMessage(messageType, p); err != nil {
			log.Println(err)
			return
		}
	}
}

func setupRoutes() {
	http.HandleFunc("/", homePage)
	http.HandleFunc("/ws", wsEndpoint)
}

func main() {
	fmt.Println("Hello World")
	setupRoutes()
	log.Fatal(http.ListenAndServe(":8080", nil))
}
