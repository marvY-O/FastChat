package main

import (
    "github.com/gorilla/mux"
    "net/http"
	"time"
	"github.com/gorilla/websocket"
	// "strings"
	"encoding/base64"
	"log"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all connections for this example
		return true
	},
}

var hub *Hub

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		log.Printf("Incoming request: %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("Outgoing response: %s %s, Duration: %s", r.Method, r.URL.Path, time.Since(startTime))
	})
}

func main() {
	
	InitAuthorization()
	InitDatabase()
	initRabbitMQ()

	log.Println("Starting Server")
	r := mux.NewRouter()

    r.Handle("/ws/chat", userHandler)

	log.Println("Server Started")

	hub = newHub()
	go hub.run()

    http.ListenAndServe(":8081", LoggingMiddleware(r))
}

type Message struct {
	Recipient string `json:"recipient"`
	Text      string `json:"text"`
}

var userHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	jwtTokenEncoded := r.FormValue("token")

	decodedBytes, err := base64.StdEncoding.DecodeString(jwtTokenEncoded)
	if err != nil {
		log.Println("Error decoding Base64:", err)
		return
	}

	// Convert bytes to a string (if the decoded content is a string)
	jwtToken := string(decodedBytes)

	if (!verifyAuth0JWT(jwtToken)){
		log.Println("Unauthorized!")
		return
	}

	user_id := getSubFromJWTToken(jwtToken)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading to WebSocket:", err)
		return
	}
	
	newClient := NewClient(hub, jwtToken, user_id, conn)
	newClient.hub.register <- newClient

	go newClient.processOutgoing();
	go newClient.processIncoming();

})

