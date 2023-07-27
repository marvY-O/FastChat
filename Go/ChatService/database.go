package main

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/bson"


)

var (
	client     *mongo.Client
	collection *mongo.Collection
	ctx        context.Context
	err        error
)

// InitDatabase initializes the MongoDB connection and collection
func InitDatabase() error {

	connectionString := "mongodb://localhost:27017"
	dbName := "mydb"
	collectionName := "chats"
	clientOptions := options.Client().ApplyURI(connectionString)

	client, err = mongo.NewClient(clientOptions)

	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		return err
	}
	collection = client.Database(dbName).Collection(collectionName)

	log.Println("Connected to MongoDB")

	return nil
}

func writeToDatabase(text TextDB, user_id1 string, user_id2 string) {
	ctx := ctx

	filter := bson.M{
		"Participants": bson.M{
			"$all": []interface{}{user_id2, user_id1},
		},
	}
	update := bson.M{
		"$push": bson.M{
			"Chats": text,
		},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Fatal("Error updating document:", err)
		// Handle the error appropriately
		return
	}
	if result.MatchedCount != 1 {
		log.Println("No matching document found.")
		// Handle the case where the document is not found
		return
	}

	// log.Println("Document updated successfully.")

}
