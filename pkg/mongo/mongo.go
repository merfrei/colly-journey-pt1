// Package mongo contains helpers for connecting and disconnecting a client
package mongo

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const cTimeout = 10 // Try max 10 seconds for client connect

// Connect is going to create a new client
func Connect(uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cTimeout*time.Second)
	defer cancel()

	client, err := mongo.
		Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("Unable to ping Mongo: %v", err)
	}

	return client, nil
}

// Disconnect is going to disconnect the client (make sure to defer this function after creating a new DB)
func Disconnect(client *mongo.Client) {
	if err := client.Disconnect(context.TODO()); err != nil {
		log.Fatal(err)
	}
}
