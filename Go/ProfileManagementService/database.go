package main

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
