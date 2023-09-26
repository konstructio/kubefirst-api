/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package db

import (
	"fmt"

	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	clusterExportsPath = "/tmp/api/cluster/export"
	clusterImportsPath = "/tmp/api/cluster/import"
)

// DeleteCluster
func (mdbcl *MongoDBClient) DeleteCluster(clusterName string) error {
	// Find
	filter := bson.D{{Key: "cluster_name", Value: clusterName}}

	// Delete
	resp, err := mdbcl.ClustersCollection.DeleteOne(mdbcl.Context, filter)
	if err != nil {
		return fmt.Errorf("error deleting cluster %s: %s", clusterName, err)
	}

	log.Infof("cluster deleted: %v", resp.DeletedCount)

	return nil
}

// GetCluster
func (mdbcl *MongoDBClient) GetCluster(clusterName string) (pkgtypes.Cluster, error) {
	// Find
	filter := bson.D{{Key: "cluster_name", Value: clusterName}}
	var result pkgtypes.Cluster
	err := mdbcl.ClustersCollection.FindOne(mdbcl.Context, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return pkgtypes.Cluster{}, fmt.Errorf("cluster not found")
		}
		return pkgtypes.Cluster{}, fmt.Errorf("error getting cluster %s: %s", clusterName, err)
	}

	return result, nil
}

// GetClusters
func (mdbcl *MongoDBClient) GetClusters() ([]pkgtypes.Cluster, error) {
	// Find all
	var results []pkgtypes.Cluster
	cursor, err := mdbcl.ClustersCollection.Find(mdbcl.Context, bson.D{})
	if err != nil {
		return []pkgtypes.Cluster{}, fmt.Errorf("error getting clusters: %s", err)
	}

	for cursor.Next(mdbcl.Context) {
		//Create a value into which the single document can be decoded
		var cl pkgtypes.Cluster
		err := cursor.Decode(&cl)
		if err != nil {
			return []pkgtypes.Cluster{}, err
		}
		results = append(results, cl)

	}
	if err := cursor.Err(); err != nil {
		return []pkgtypes.Cluster{}, err
	}

	cursor.Close(mdbcl.Context)

	return results, nil
}

// InsertCluster
func (mdbcl *MongoDBClient) InsertCluster(cl pkgtypes.Cluster) error {
	filter := bson.D{{Key: "cluster_name", Value: cl.ClusterName}}
	var result pkgtypes.Cluster
	err := mdbcl.ClustersCollection.FindOne(mdbcl.Context, filter).Decode(&result)
	if err != nil {
		// This error means your query did not match any documents.
		if err == mongo.ErrNoDocuments {
			// Create if entry does not exist
			insert, err := mdbcl.ClustersCollection.InsertOne(mdbcl.Context, cl)
			if err != nil {
				return fmt.Errorf("error inserting cluster %s: %s", cl.ClusterName, err)
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
	filter := bson.D{{Key: "cluster_name", Value: clusterName}}
	var result pkgtypes.Cluster
	err := mdbcl.ClustersCollection.FindOne(mdbcl.Context, filter).Decode(&result)
	if err != nil {
		return fmt.Errorf("error finding cluster %s: %s", clusterName, err)
	}

	// Update
	filter = bson.D{{Key: "_id", Value: result.ID}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: field, Value: value}}}}
	_, err = mdbcl.ClustersCollection.UpdateOne(mdbcl.Context, filter, update)
	if err != nil {
		return fmt.Errorf("error updating cluster %s: %s", clusterName, err)
	}

	return nil
}
