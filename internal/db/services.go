/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package db

import (
	"fmt"

	"github.com/kubefirst/kubefirst-api/internal/types"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// CreateClusterServiceList adds an entry for a cluster to the service list
func (mdbcl *MongoDBClient) CreateClusterServiceList(cl *types.Cluster) error {
	filter := bson.D{{"cluster_name", cl.ClusterName}}
	var result types.Cluster
	err := mdbcl.ServicesCollection.FindOne(mdbcl.Context, filter).Decode(&result)
	if err != nil {
		// This error means your query did not match any documents.
		if err == mongo.ErrNoDocuments {
			// Create if entry does not exist
			_, err := mdbcl.ServicesCollection.InsertOne(mdbcl.Context, types.ClusterServiceList{
				ClusterName: cl.ClusterName,
				Services:    []types.Service{},
			})
			if err != nil {
				return fmt.Errorf("error inserting cluster service list for cluster %s: %s", cl.ClusterName, err)
			}
		}
	} else {
		log.Infof("cluster service list record for %s already exists - skipping", cl.ClusterName)
	}

	return nil
}

// DeleteClusterServiceListEntry removes a service entry from a cluster's service list
func (mdbcl *MongoDBClient) DeleteClusterServiceListEntry(clusterName string, def *types.Service) error {
	// Find
	filter := bson.D{{"cluster_name", clusterName}}

	// Update
	update := bson.M{"$pull": bson.M{"services": def}}
	resp, err := mdbcl.ServicesCollection.UpdateOne(mdbcl.Context, filter, update)
	if err != nil {
		return fmt.Errorf("error updating cluster service list for cluster %s: %s", clusterName, err)
	}

	log.Infof("cluster service list updated: %v", resp.ModifiedCount)

	return nil
}

// GetService returns a single service associated with a given cluster
func (mdbcl *MongoDBClient) GetService(clusterName string, serviceName string) (types.Service, error) {
	// Find
	filter := bson.D{{"cluster_name", clusterName}}
	var result types.ClusterServiceList
	err := mdbcl.ServicesCollection.FindOne(mdbcl.Context, filter).Decode(&result)
	if err != nil {
		return types.Service{}, fmt.Errorf("error getting service %s for cluster %s: %s", serviceName, clusterName, err)
	}

	for _, service := range result.Services {
		if service.Name == serviceName {
			return service, nil
		}
	}

	return types.Service{}, fmt.Errorf("could not find service %s for cluster %s", serviceName, clusterName)
}

// GetServices returns services associated with a given cluster
func (mdbcl *MongoDBClient) GetServices(clusterName string) (types.ClusterServiceList, error) {
	// Find
	filter := bson.D{{"cluster_name", clusterName}}
	var result types.ClusterServiceList
	err := mdbcl.ServicesCollection.FindOne(mdbcl.Context, filter).Decode(&result)
	if err != nil {
		return types.ClusterServiceList{}, fmt.Errorf("error getting service list for cluster %s: %s", clusterName, err)
	}

	return result, nil
}

// InsertClusterServiceListEntry appends a service entry for a cluster's service list
func (mdbcl *MongoDBClient) InsertClusterServiceListEntry(clusterName string, def *types.Service) error {
	// Find
	filter := bson.D{{"cluster_name", clusterName}}

	// Update
	update := bson.M{"$push": bson.M{"services": def}}
	_, err := mdbcl.ServicesCollection.UpdateOne(mdbcl.Context, filter, update)
	if err != nil {
		return fmt.Errorf("error updating cluster service list for cluster %s: %s", clusterName, err)
	}

	return nil
}
