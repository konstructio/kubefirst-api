/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package secrets

import (
	"encoding/json"
	"fmt"

	"github.com/kubefirst/kubefirst-api/pkg/k8s"
	"github.com/kubefirst/kubefirst-api/pkg/types"
	log "github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"k8s.io/client-go/kubernetes"
)

const KUBEFIRST_SERVICES_SECRET_NAME = "kubefirst-services"

// CreateClusterServiceList adds an entry for a cluster to the service list
func CreateClusterServiceList(clientSet *kubernetes.Clientset, clusterName string) error {
	filter := bson.D{{Key: "cluster_name", Value: clusterName}}
	var result types.Cluster
	err := mdbcl.ServicesCollection.FindOne(mdbcl.Context, filter).Decode(&result)
	if err != nil {
		// This error means your query did not match any documents.
		if err == mongo.ErrNoDocuments {
			// Create if entry does not exist
			_, err := mdbcl.ServicesCollection.InsertOne(mdbcl.Context, types.ClusterServiceList{
				ClusterName: clusterName,
				Services:    []types.Service{},
			})
			if err != nil {
				return fmt.Errorf("error inserting cluster service list for cluster %s: %s", clusterName, err)
			}
		}
	} else {
		log.Info().Msgf("cluster service list record for %s already exists - skipping", clusterName)
	}

	return nil
}

// DeleteClusterServiceListEntry removes a service entry from a cluster's service list
func DeleteClusterServiceListEntry(clientSet *kubernetes.Clientset, clusterName string, def *types.Service) error {
	// Find
	clusterServices, err := GetServices(clientSet, clusterName)
	filteredServiceList := []types.Service{}

	for _, service := range clusterServices.Services {
		if service.Name != def.Name {
			filteredServiceList = append(filteredServiceList, service)
		}
	}

	bytes, err := json.Marshal(filteredServiceList)
	secretValuesMap, _ := ParseJSONToMap(string(bytes))

	err = k8s.UpdateSecretV2(clientSet, "kubefirst", KUBEFIRST_SERVICES_SECRET_NAME, secretValuesMap)
	if err != nil {
		return fmt.Errorf("error deleting service list entry %s: %s", def.Name, err)
	}

	log.Info().Msgf("service deleted: %v", def.Name)

	return nil
}

// GetService returns a single service associated with a given cluster
func GetService(clientSet *kubernetes.Clientset, clusterName string, serviceName string) (types.Service, error) {
	// Find
	clusterServices, _ := GetServices(clientSet, clusterName)

	for _, service := range clusterServices.Services {
		if service.Name == serviceName {
			return service, nil
		}
	}

	return types.Service{}, fmt.Errorf("could not find service %s for cluster %s", serviceName, clusterName)
}

// GetServices returns services associated with a given cluster
func GetServices(clientSet *kubernetes.Clientset, clusterName string) (types.ClusterServiceList, error) {
	clusterServices := types.ClusterServiceList{}

	kubefirstSecrets, err := k8s.ReadSecretV2(clientSet, "kubefirst", KUBEFIRST_SERVICES_SECRET_NAME)

	jsonString, err := MapToStructuredJSON(kubefirstSecrets)

	jsonData, err := json.Marshal(jsonString)
	if err != nil {
		return clusterServices, fmt.Errorf("error marshalling json: %s", err)
	}

	err = json.Unmarshal([]byte(jsonData), &clusterServices)
	if err != nil {
		return clusterServices, fmt.Errorf("unable to cast environment: %s", err)
	}

	return clusterServices, nil
}

// InsertClusterServiceListEntry appends a service entry for a cluster's service list
func InsertClusterServiceListEntry(clientSet *kubernetes.Clientset, clusterName string, def *types.Service) error {
	// Find
	clusterServices, err := GetServices(clientSet, clusterName)
	clusterServices.Services = append(clusterServices.Services, *def)

	bytes, err := json.Marshal(clusterServices)
	secretValuesMap, _ := ParseJSONToMap(string(bytes))

	err = k8s.UpdateSecretV2(clientSet, "kubefirst", KUBEFIRST_SERVICES_SECRET_NAME, secretValuesMap)
	if err != nil {
		return fmt.Errorf("error adding service list entry %s: %s", def.Name, err)
	}

	log.Info().Msgf("service added: %v", def.Name)

	return nil
}
