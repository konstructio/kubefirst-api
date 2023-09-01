/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package db

import (
	"context"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBClient struct {
	Client                  *mongo.Client
	Context                 context.Context
	ClustersCollection      *mongo.Collection
	GitopsCatalogCollection *mongo.Collection
	ServicesCollection      *mongo.Collection
}

var Client = Connect()

// 1 Client, Mongo not ready

// Connect
func Connect() *MongoDBClient {
	var connString string
	var clientOptions *options.ClientOptions

	ctx := context.Background()

	switch os.Getenv("MONGODB_HOST_TYPE") {
	case "atlas":
		serverAPI := options.ServerAPI(options.ServerAPIVersion1)
		connString = fmt.Sprintf("mongodb+srv://%s:%s@%s",
			os.Getenv("MONGODB_USERNAME"),
			os.Getenv("MONGODB_PASSWORD"),
			os.Getenv("MONGODB_HOST"),
		)
		clientOptions = options.Client().ApplyURI(connString).SetServerAPIOptions(serverAPI)
	case "local":
		connString = fmt.Sprintf("mongodb://%s:%s@%s/?authSource=admin",
			os.Getenv("MONGODB_USERNAME"),
			os.Getenv("MONGODB_PASSWORD"),
			os.Getenv("MONGODB_HOST"),
		)
		clientOptions = options.Client().ApplyURI(connString)
	}

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("could not create mongodb client: %s", err)
	}

	cl := MongoDBClient{
		Client:                  client,
		Context:                 ctx,
		ClustersCollection:      client.Database("api").Collection("clusters"),
		GitopsCatalogCollection: client.Database("api").Collection("gitops-catalog"),
		ServicesCollection:      client.Database("api").Collection("services"),
	}

	return &cl
}

// TestDatabaseConnection
func (mdbcl *MongoDBClient) TestDatabaseConnection(silent bool) error {
	err := mdbcl.Client.Database("admin").RunCommand(mdbcl.Context, bson.D{{Key: "ping", Value: 1}}).Err()
	if err != nil {
		log.Fatalf("error connecting to mongodb: %s", err)
	}
	if !silent {
		log.Infof("connected to mongodb host %s", os.Getenv("MONGODB_HOST"))
	}

	return nil
}
