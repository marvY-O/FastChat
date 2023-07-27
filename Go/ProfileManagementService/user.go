package main

import (
    "encoding/json"
	"fmt"
	"io/ioutil"
    "net/http"
	"strings"
	"go.mongodb.org/mongo-driver/bson"
	"log"
)

type SearchPerson struct {
    Email string
}

type SearchPersonResponse struct {
	User_id string
    Email string
	Name string
	Picture string
}

var fetch_by_email = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
	email := r.FormValue("email")
	if (len(email) == 0){
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	url := auth0api.Issuer+"api/v2/users-by-email?email="+email

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("authorization", "Bearer "+auth0api.MgmtAccessToken+"")

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	// fmt.Println(string(body))

	var response []map[string]interface{}
	err := json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("Error:", err)
	}

	var payload []byte;

	if (len(response) == 0){
		w.WriteHeader(http.StatusNotFound);
		return;
	} else{
		for i := 0; i < len(response); i++ {
			if (strings.HasPrefix(response[i]["user_id"].(string), "google")){
				payload, _ = json.Marshal(response[i]);
				break;
			}
		}
		if (len(payload) == 0){
			payload, _ = json.Marshal(response[0]);
		}
	}

	
	w.Header().Set("Content-Type", "application/json")
    w.Write([] byte(payload))
})

var fetch_by_id = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
	user_id := r.FormValue("user_id")
	if (len(user_id) == 0){
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	url := auth0api.Issuer+"api/v2/users/"+user_id

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("authorization", "Bearer "+auth0api.MgmtAccessToken)

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	var response map[string]interface{}
	err := json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("Error:", err)
	}

	// var payload []byte;

	payload, _ := json.Marshal(response)
	
	w.Header().Set("Content-Type", "application/json")
    w.Write([] byte(payload))
})


type Chat struct {
	Participants []string `bson:"Participants"`
	Chats        []struct {
		Text_id string `bson:Text_id`
		Sender_id  string `bson:"Sender_id"`
		Timestamp int64  `bson:"Timestamp"`
		Text      string `bson:"Text"`
	} `bson:"Chats"`
}


var fetchChats = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	authorizationHeader := r.Header.Get("Authorization")
	if authorizationHeader == "" {
		http.Error(w, "Authorization header missing", http.StatusUnauthorized)
		return
	}
	if !strings.HasPrefix(authorizationHeader, "Bearer ") {
		http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
		return
	}
	tokenString := strings.TrimPrefix(authorizationHeader, "Bearer ")

	specificParticipant := getSubFromJWTToken(tokenString)

	ctx := ctx

	filter := bson.M{"Participants": specificParticipant}

	cur, err := client.Database("mydb").Collection("chats").Find(ctx, filter)
	if err != nil {
		log.Fatal("Error finding documents:", err)
		http.Error(w, "Error finding documents", http.StatusInternalServerError)
		return
	}
	defer cur.Close(ctx)

	var chats []Chat
	for cur.Next(ctx) {
		var chat Chat
		if err := cur.Decode(&chat); err != nil {
			log.Fatal("Error decoding document:", err)
			http.Error(w, "Error decoding document", http.StatusInternalServerError)
			return
		}
		chats = append(chats, chat)
	}

	if err := cur.Err(); err != nil {
		log.Fatal("Error iterating through cursor:", err)
		http.Error(w, "Error iterating through cursor", http.StatusInternalServerError)
		return
	}

	payload, err := json.Marshal(chats)
	if err != nil {
		log.Fatal("Error marshaling chats to JSON:", err)
		http.Error(w, "Error marshaling chats to JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
})
