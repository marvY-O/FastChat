package main

import (
    "github.com/gorilla/mux"
	"github.com/rs/cors"
    "net/http"
	"time"
	"log"
)

type Response struct {
    Message string `json:"message"`
}


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
	
	log.Println("Starting Server")
	r := mux.NewRouter()

    r.Handle("/api/users/fetch/email", jwtMiddleware.Handler(fetch_by_email)).Methods("GET")
    r.Handle("/api/users/fetch/id", jwtMiddleware.Handler(fetch_by_id)).Methods("GET")
    r.Handle("/api/chats/fetch", jwtMiddleware.Handler(fetchChats)).Methods("GET")


    corsWrapper := cors.New(cors.Options {
        AllowedMethods: [] string {
            "GET", "POST",
        },
        AllowedHeaders: [] string {
            "Content-Type", "Origin", "Accept", "*",
        },
    })

	corsHandler := corsWrapper.Handler(LoggingMiddleware(r))

	log.Println("Server Started")

    http.ListenAndServe(":8080", corsHandler)
}




