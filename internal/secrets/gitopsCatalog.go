/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package secrets

import (
	"encoding/json"
	"fmt"

	"github.com/konstructio/kubefirst-api/internal/gitopsCatalog"
	"github.com/konstructio/kubefirst-api/internal/k8s"
	"github.com/konstructio/kubefirst-api/pkg/types"
	log "github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const kubefirstCatalogSecretName = "kubefirst-catalog"

// CreateGitopsCatalogApps
func CreateGitopsCatalogApps(clientSet *kubernetes.Clientset, catalogApps types.GitopsCatalogApps) error {
	bytes, err := json.Marshal(catalogApps)
	if err != nil {
		return fmt.Errorf("error marshalling json: %w", err)
	}

	secretValuesMap, err := ParseJSONToMap(string(bytes))
	if err != nil {
		return fmt.Errorf("error parsing json to map: %w", err)
	}

	secretToCreate := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubefirstCatalogSecretName,
			Namespace: "kubefirst",
		},
		Data: secretValuesMap,
	}

	if err := k8s.CreateSecretV2(clientSet, secretToCreate); err != nil {
		return fmt.Errorf("error creating gitops catalog secret: %w", err)
	}

	return nil
}

// GetGitopsCatalogApps
func GetGitopsCatalogApps(clientSet *kubernetes.Clientset) (types.GitopsCatalogApps, error) {
	catalogApps := types.GitopsCatalogApps{}

	kubefirstSecrets, err := k8s.ReadSecretV2Old(clientSet, "kubefirst", kubefirstCatalogSecretName)
	if err != nil {
		return catalogApps, fmt.Errorf("error reading kubernetes secret: %w", err)
	}

	jsonString, err := MapToStructuredJSON(kubefirstSecrets)
	if err != nil {
		return catalogApps, fmt.Errorf("error parsing json: %w", err)
	}

	jsonData, err := json.Marshal(jsonString)
	if err != nil {
		return catalogApps, fmt.Errorf("error marshalling json: %w", err)
	}

	err = json.Unmarshal(jsonData, &catalogApps)
	if err != nil {
		return catalogApps, fmt.Errorf("unable to cast catalog: %w", err)
	}

	return catalogApps, nil
}

// GetGitopsCatalogAppsByCloudProvider
func GetGitopsCatalogAppsByCloudProvider(clientSet *kubernetes.Clientset, cloudProvider string, gitProvider string) (types.GitopsCatalogApps, error) {
	result, err := GetGitopsCatalogApps(clientSet)
	if err != nil {
		return result, fmt.Errorf("error getting gitops catalog apps: %w", err)
	}

	filteredApps := []types.GitopsCatalogApp{}

	for _, app := range result.Apps {
		if !slices.Contains(app.CloudDenylist, cloudProvider) && !slices.Contains(app.GitDenylist, gitProvider) {
			filteredApps = append(filteredApps, app)
		}
	}

	result.Apps = filteredApps

	return result, nil
}

// UpdateGitopsCatalogApps
func UpdateGitopsCatalogApps(clientSet *kubernetes.Clientset) error {
	mpapps, err := gitopsCatalog.ReadActiveApplications()
	if err != nil {
		log.Error().Msgf("error reading gitops catalog apps at startup: %s", err)
	}

	catalogApps, err := GetGitopsCatalogApps(clientSet)
	if err != nil {
		log.Error().Msgf("error fetching gitops catalog apps: %s", err)
		return fmt.Errorf("error fetching gitops catalog apps: %w", err)
	}

	// If no apps are found, create the GitOps catalog apps
	if len(catalogApps.Apps) == 0 {
		err = CreateGitopsCatalogApps(clientSet, mpapps)
		if err != nil {
			log.Error().Msgf("error creating gitops catalog apps secret: %s", err)
			return fmt.Errorf("error creating gitops catalog apps secret: %w", err)
		}
	} else {
		catalogApps.Apps = mpapps.Apps

		bytes, err := json.Marshal(catalogApps)
		if err != nil {
			return fmt.Errorf("error marshalling json: %w", err)
		}

		secretValuesMap, err := ParseJSONToMap(string(bytes))
		if err != nil {
			return fmt.Errorf("error parsing json to map: %w", err)
		}

		err = k8s.UpdateSecretV2(clientSet, "kubefirst", kubefirstCatalogSecretName, secretValuesMap)
		if err != nil {
			return fmt.Errorf("error creating kubernetes secret: %w", err)
		}
	}

	return nil
}
