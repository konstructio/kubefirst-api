/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package secrets

import (
	"encoding/json"
	"fmt"

	"github.com/konstructio/kubefirst-api/internal/k8s"
	pkgtypes "github.com/konstructio/kubefirst-api/pkg/types"
	log "github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	secretName    = "kubefirst-clusters"
	clusterPrefix = "kubefirst-cluster"
)

// DeleteCluster
func DeleteCluster(clientSet kubernetes.Interface, clusterName string) error {
	err := DeleteSecretReference(clientSet, secretName, clusterName)
	if err != nil {
		return fmt.Errorf("error deleting cluster %s reference", clusterName)
	}

	err = k8s.DeleteSecretV2(clientSet, "kubefirst", fmt.Sprintf("%s-%s", clusterPrefix, clusterName))
	if err != nil {
		return fmt.Errorf("error deleting cluster %s: %w", clusterName, err)
	}

	log.Info().Msgf("cluster deleted: %v", clusterName)

	return nil
}

func isMapEmpty(m map[string]interface{}) bool {
	return len(m) == 0
}

type ClusterNotFoundError struct {
	ClusterName string
}

func (e *ClusterNotFoundError) Error() string {
	return fmt.Sprintf("cluster %q not found", e.ClusterName)
}

func (e *ClusterNotFoundError) Is(target error) bool {
	_, ok := target.(*ClusterNotFoundError)
	return ok
}

// GetCluster
func GetCluster(clientSet kubernetes.Interface, clusterName string) (*pkgtypes.Cluster, error) {
	cluster := pkgtypes.Cluster{}

	clusterSecret, err := k8s.ReadSecretV2Old(clientSet, "kubefirst", fmt.Sprintf("%s-%s", clusterPrefix, clusterName))
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, &ClusterNotFoundError{ClusterName: clusterName}
		}

		return nil, fmt.Errorf("secret not found: %w", err)
	}

	if isMapEmpty(clusterSecret) {
		return nil, &ClusterNotFoundError{ClusterName: clusterName}
	}

	jsonString, err := MapToStructuredJSON(clusterSecret)
	if err != nil {
		return nil, fmt.Errorf("error mapping to structured json: %w", err)
	}

	jsonData, err := json.Marshal(jsonString)
	if err != nil {
		return nil, fmt.Errorf("error marshalling json: %w", err)
	}

	err = json.Unmarshal(jsonData, &cluster)
	if err != nil {
		return nil, fmt.Errorf("unable to cast cluster: %w", err)
	}

	return &cluster, nil
}

// GetCluster
func GetClusters(clientSet kubernetes.Interface) ([]pkgtypes.Cluster, error) {
	clusterList := []pkgtypes.Cluster{}
	clusterReferenceList, err := GetSecretReference(clientSet, secretName)
	if err != nil {
		return nil, fmt.Errorf("unable to get secret cluster reference: %w", err)
	}

	for _, clusterName := range clusterReferenceList.List {
		cluster, err := GetCluster(clientSet, clusterName)
		if err != nil {
			return nil, fmt.Errorf("unable to get cluster %s: %w", clusterName, err)
		}

		if cluster.ClusterName != "" {
			clusterList = append(clusterList, *cluster)
		}
	}

	return clusterList, nil
}

// InsertCluster
func InsertCluster(clientSet kubernetes.Interface, cl pkgtypes.Cluster) error {
	_, err := GetSecretReference(clientSet, secretName)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("unable to get secret cluster reference: %w", err)
	}

	if apierrors.IsNotFound(err) {
		secretReference := pkgtypes.SecretListReference{
			Name: "clusters",
			List: []string{cl.ClusterName},
		}
		if err := UpsertSecretReference(clientSet, secretName, secretReference); err != nil {
			return fmt.Errorf("when inserting cluster: error creating secret reference: %w", err)
		}
	}

	if err := AddSecretReferenceItem(clientSet, secretName, cl.ClusterName); err != nil {
		return fmt.Errorf("when inserting cluster: error adding secret reference item: %w", err)
	}

	bytes, err := json.Marshal(cl)
	if err != nil {
		return fmt.Errorf("error marshalling json: %w", err)
	}

	secretValuesMap, err := ParseJSONToMap(string(bytes))
	if err != nil {
		return fmt.Errorf("error parsing json to map: %w", err)
	}

	secretToCreate := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", clusterPrefix, cl.ClusterName),
			Namespace: "kubefirst",
		},
		Data: secretValuesMap,
	}

	err = k8s.CreateSecretV2(clientSet, secretToCreate)
	if err != nil {
		return fmt.Errorf("error creating kubernetes secret: %w", err)
	}

	return nil
}

// UpdateCluster
func UpdateCluster(clientSet kubernetes.Interface, cluster pkgtypes.Cluster) error {
	bytes, err := json.Marshal(cluster)
	if err != nil {
		return fmt.Errorf("error marshalling json: %w", err)
	}

	secretValuesMap, err := ParseJSONToMap(string(bytes))
	if err != nil {
		return fmt.Errorf("error parsing json to map: %w", err)
	}

	err = k8s.UpdateSecretV2(clientSet, "kubefirst", fmt.Sprintf("%s-%s", clusterPrefix, cluster.ClusterName), secretValuesMap)
	if err != nil {
		return fmt.Errorf("error updating kubernetes secret: %w", err)
	}

	return nil
}
