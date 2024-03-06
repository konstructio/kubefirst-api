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
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	log "github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const KUBEFIRST_CLUSTERS_SECRET_NAME = "kubefirst-clusters"

// DeleteCluster
func DeleteCluster(clientSet *kubernetes.Clientset, clusterName string) error {
	clusterList, err := GetClusters(clientSet)
	filteredClusterList := []pkgtypes.Cluster{}

	for _, cluster := range clusterList {
		if cluster.ClusterName != clusterName {
			filteredClusterList = append(filteredClusterList, cluster)
		}
	}

	bytes, err := json.Marshal(filteredClusterList)
	secretValuesMap, _ := ParseJSONToMap(string(bytes))

	err = k8s.UpdateSecretV2(clientSet, "kubefirst", KUBEFIRST_CLUSTERS_SECRET_NAME, secretValuesMap)
	if err != nil {
		return fmt.Errorf("error updating clusters %s: %s", clusterName, err)
	}

	log.Info().Msgf("cluster deleted: %v", clusterName)

	return nil
}

// GetCluster
func GetCluster(clientSet *kubernetes.Clientset, clusterName string) (pkgtypes.Cluster, error) {
	clusterList, _ := GetClusters(clientSet)
	cluster := pkgtypes.Cluster{}
	for _, clusterItem := range clusterList {
		if clusterItem.ClusterName == clusterName {
			cluster = clusterItem
		}
	}

	return cluster, nil
}

// GetCluster
func GetClusters(clientSet *kubernetes.Clientset) ([]pkgtypes.Cluster, error) {
	clusterList := []pkgtypes.Cluster{}

	kubefirstSecrets, err := k8s.ReadSecretV2(clientSet, "kubefirst", KUBEFIRST_CLUSTERS_SECRET_NAME)

	jsonString, err := MapToStructuredJSON(kubefirstSecrets)

	jsonData, err := json.Marshal(jsonString)
	if err != nil {
		return clusterList, fmt.Errorf("error marshalling json: %s", err)
	}

	err = json.Unmarshal([]byte(jsonData), &clusterList)
	if err != nil {
		return clusterList, fmt.Errorf("unable to cast cluster: %s", err)
	}

	return clusterList, nil
}

// InsertCluster
func InsertCluster(clientSet *kubernetes.Clientset, cl pkgtypes.Cluster) error {
	clusters, _ := GetClusters(clientSet)

	if len(clusters) == 0 {
		clusters = append(clusters, cl)
		bytes, _ := json.Marshal(clusters)
		secretValuesMap, _ := ParseJSONToMap(string(bytes))

		secretToCreate := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      KUBEFIRST_CLUSTERS_SECRET_NAME,
				Namespace: "kubefirst",
			},
			Data: secretValuesMap,
		}

		err := k8s.CreateSecretV2(clientSet, secretToCreate)

		if err != nil {
			return fmt.Errorf("error creating kubernetes secret: %s", err)
		}

		return nil
	}

	clusterList := append(clusters, cl)
	bytes, _ := json.Marshal(clusterList)
	secretValuesMap, _ := ParseJSONToMap(string(bytes))

	err := k8s.UpdateSecretV2(clientSet, "kubefirst", KUBEFIRST_CLUSTERS_SECRET_NAME, secretValuesMap)

	if err != nil {
		return fmt.Errorf("error creating kubernetes secret: %s", err)
	}

	return nil
}

// UpdateCluster
func UpdateCluster(clientSet *kubernetes.Clientset, cluster pkgtypes.Cluster) error {
	clusters, _ := GetClusters(clientSet)

	for _, clusterItem := range clusters {
		if clusterItem.ClusterName == cluster.ClusterName {
			clusterItem = cluster
		}
	}

	bytes, _ := json.Marshal(clusters)
	secretValuesMap, _ := ParseJSONToMap(string(bytes))

	err := k8s.UpdateSecretV2(clientSet, "kubefirst", KUBEFIRST_CLUSTERS_SECRET_NAME, secretValuesMap)

	if err != nil {
		return fmt.Errorf("error creating kubernetes secret: %s", err)
	}

	return nil
}
