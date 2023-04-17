/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package db

import (
	"context"

	"github.com/kubefirst/kubefirst-api/internal/types"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBClient struct {
	Client     *mongo.Client
	Collection *mongo.Collection
	Context    context.Context
}

// InitDatabase
func (mdbcl *MongoDBClient) InitDatabase() error {
	mdbcl.Context = context.Background()

	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017/")
	client, err := mongo.Connect(mdbcl.Context, clientOptions)
	if err != nil {
		return err
	}

	err = client.Ping(mdbcl.Context, nil)
	if err != nil {
		return err
	}

	mdbcl.Client = client
	mdbcl.Collection = client.Database("api").Collection("clusters")

	return nil
}

// CRUD

// DeleteCluster
func (mdbcl *MongoDBClient) DeleteCluster(clusterName string) error {
	// Find
	filter := bson.D{{"cluster_name", clusterName}}

	// Delete
	resp, err := mdbcl.Collection.DeleteOne(mdbcl.Context, filter)
	if err != nil {
		return err
	}

	log.Infof("cluster deleted: %v", resp.DeletedCount)

	return nil
}

// GetCluster
func (mdbcl *MongoDBClient) GetCluster(clusterName string) (types.Cluster, error) {
	// Find
	filter := bson.D{{"cluster_name", clusterName}}
	var result types.Cluster
	err := mdbcl.Collection.FindOne(mdbcl.Context, filter).Decode(&result)
	if err != nil {
		return types.Cluster{}, err
	}

	return result, nil
}

// GetClusters
func (mdbcl *MongoDBClient) GetClusters() ([]types.Cluster, error) {
	// Find all
	var results []types.Cluster
	cursor, err := mdbcl.Collection.Find(mdbcl.Context, bson.D{})
	if err != nil {
		return []types.Cluster{}, err
	}

	for cursor.Next(mdbcl.Context) {
		//Create a value into which the single document can be decoded
		var cl types.Cluster
		err := cursor.Decode(&cl)
		if err != nil {
			return []types.Cluster{}, err
		}
		results = append(results, cl)

	}
	if err := cursor.Err(); err != nil {
		return []types.Cluster{}, err
	}

	cursor.Close(mdbcl.Context)

	return results, nil
}

// InsertCluster
func (mdbcl *MongoDBClient) InsertCluster(cl types.Cluster) error {
	filter := bson.D{{"cluster_name", cl.ClusterName}}
	var result types.Cluster
	err := mdbcl.Collection.FindOne(mdbcl.Context, filter).Decode(&result)
	if err != nil {
		// This error means your query did not match any documents.
		if err == mongo.ErrNoDocuments {
			// Create if entry does not exist
			insert, err := mdbcl.Collection.InsertOne(mdbcl.Context, cl)
			if err != nil {
				return err
			}
			log.Info(insert)
		}
	} else {
		log.Infof("cluster record for %s already exists - skipping", cl.ClusterName)
	}

	return nil
}

// UpdateCluster
func (mdbcl *MongoDBClient) UpdateCluster(clusterName string, field string, value interface{}) error {
	// Find
	filter := bson.D{{"cluster_name", clusterName}}
	var result types.Cluster
	err := mdbcl.Collection.FindOne(mdbcl.Context, filter).Decode(&result)
	if err != nil {
		return err
	}

	// Update
	filter = bson.D{{"_id", result.ID}}
	update := bson.D{{"$set", bson.D{{field, value}}}}
	resp, err := mdbcl.Collection.UpdateOne(mdbcl.Context, filter, update)
	if err != nil {
		return err
	}

	log.Infof("cluster updated: %v", resp.ModifiedCount)

	return nil
}
