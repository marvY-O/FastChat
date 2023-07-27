package main

import (
    "encoding/json"
    "errors"
	"crypto/subtle"
	"fmt"
	"io/ioutil"
    "github.com/auth0/go-jwt-middleware"
    "github.com/form3tech-oss/jwt-go"
    "github.com/gorilla/mux"
	"github.com/rs/cors"
    "net/http"
	"time"
	"strings"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"context"
	"log"
)

var auth0api Auth0Cred
var collection *mongo.Collection
var (
    client *mongo.Client
    ctx    context.Context
	err 	error
)

type Response struct {
    Message string `json:"message"`
}

type Jwks struct {
    Keys[] JSONWebKeys `json:"keys"`
}

type JSONWebKeys struct {
    Kty string `json:"kty"`
    Kid string `json:"kid"`
    Use string `json:"use"`
    N string `json:"n"`
    E string `json:"e"`
    X5c[] string `json:"x5c"`
}

type Auth0Cred struct {
    Audience string
    Issuer string
	ClientID string
	MgmtAccessToken string
}


func VerifyAudience(m jwt.MapClaims, cmp string, req bool) bool {
	var aud []string
	switch v := m["aud"].(type) {
	case string:
		aud = append(aud, v)
	case []string:
		aud = v
	case []interface{}:
		for _, a := range v {
			vs, ok := a.(string)
			if !ok {
				return false
			}
			aud = append(aud, vs)
			break;
		}
	}
	return verifyAud(aud, cmp, req)
}

func verifyAud(aud []string, cmp string, required bool) bool {
	if len(aud) == 0 {
		return !required
	}
	// use a var here to keep constant time compare when looping over a number of claims
	result := false

	var stringClaims string
	for _, a := range aud {
		if subtle.ConstantTimeCompare([]byte(a), []byte(cmp)) != 0 {
			result = true
		}
		stringClaims = stringClaims + a
	}

	// case where "" is sent in one or many aud claims
	if len(stringClaims) == 0 {
		return !required
	}

	return result
}

func getPemCert(token *jwt.Token) (string, error) {
    cert := ""
    resp, err := http.Get(auth0api.Issuer + ".well-known/jwks.json")

    if err != nil {
        return cert, err
    }
    defer resp.Body.Close()

    var jwks = Jwks{}
    err = json.NewDecoder(resp.Body).Decode(&jwks)

    if err != nil {
        return cert, err
    }

    for k, _ := range jwks.Keys {
        if token.Header["kid"] == jwks.Keys[k].Kid {
            cert = "-----BEGIN CERTIFICATE-----\n" + jwks.Keys[k].X5c[0] + "\n-----END CERTIFICATE-----"
        }
    }

    if cert == "" {
        err := errors.New("Unable to find appropriate key.")
        return cert, err
    }

    return cert, nil
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// Log the incoming request
		log.Printf("Incoming request: %s %s", r.Method, r.URL.Path)

		// Call the next handler in the chain
		next.ServeHTTP(w, r)

		// Log the outgoing response
		log.Printf("Outgoing response: %s %s, Duration: %s", r.Method, r.URL.Path, time.Since(startTime))
	})
}

func main() {
	plan, _ := os.ReadFile("../auth0api.json")
	json.Unmarshal(plan, &auth0api)
	// fmt.Println(auth0api)
	jwtMiddleware := jwtmiddleware.New(jwtmiddleware.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			// Verify 'aud' claim
			aud := auth0api.Audience
			checkAud := VerifyAudience(token.Claims.(jwt.MapClaims), aud, false)
			if !checkAud {
				return token, errors.New("Invalid audience.")
			}

			// Verify 'iss' claim
			iss := auth0api.Issuer
			checkIss := token.Claims.(jwt.MapClaims).VerifyIssuer(iss, false)
			if !checkIss {
				return token, errors.New("Invalid issuer.")
			}

			cert, err := getPemCert(token)
			if err != nil {
				panic(err.Error())
			}

			result, _ := jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
			return result, nil
		},
		SigningMethod: jwt.SigningMethodRS256,
	})

	connectionString := "mongodb://localhost:27017"
	dbName := "mydb"
	collectionName := "chats"

	// Create a MongoDB client
	client, err = mongo.NewClient(options.Client().ApplyURI(connectionString))
	if err != nil {
		log.Fatal("Error creating MongoDB client:", err)
	} else{
		log.Println("MongoDb initializing...");
	}
	//dfd
	clientOptions := options.Client().ApplyURI(connectionString)
    clientOptions.SetMaxPoolSize(100) // Set the maximum pool size to 100

    var err error
    client, err = mongo.NewClient(clientOptions)
    if err != nil {
        log.Fatal("Error creating MongoDB client:", err)
        return
    }

    // Create a context that will be used for the entire application
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Connect to the MongoDB server
    err = client.Connect(ctx)
    if err != nil {
        log.Fatal("Error connecting to MongoDB:", err)
        return
    }
    defer client.Disconnect(ctx)

	//fdsfds
	collection = client.Database(dbName).Collection(collectionName)

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

func getSubFromJWTToken(tokenString string) string {
	token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return nil, nil // Since the token is already verified, we don't need to re-verify it here.
	})
	claims, _ := token.Claims.(jwt.MapClaims)
	sub, _ := claims["sub"].(string)
	return sub
}

type Chat struct {
	Participants []string `bson:"Participants"`
	Chats        []struct {
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

	// Use the global context for the handler
	ctx := ctx

	// Use the global client for the handler
	// No need to connect or disconnect here since it's already managed in the main function
	// Use 'client' directly instead of creating a new one

	// Define the filter to find documents that contain the specific value in the "Participants" array
	filter := bson.M{"Participants": specificParticipant}

	// Perform the find operation
	cur, err := client.Database("mydb").Collection("chats").Find(ctx, filter)
	if err != nil {
		log.Fatal("Error finding documents:", err)
		http.Error(w, "Error finding documents", http.StatusInternalServerError)
		return
	}
	defer cur.Close(ctx)

	// Iterate through the result cursor and decode the documents
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

	// Convert the chats to JSON and write the response
	payload, err := json.Marshal(chats)
	if err != nil {
		log.Fatal("Error marshaling chats to JSON:", err)
		http.Error(w, "Error marshaling chats to JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
})




