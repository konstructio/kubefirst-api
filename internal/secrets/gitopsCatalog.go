/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package secrets

import (
	"encoding/json"
	"fmt"

	"github.com/kubefirst/kubefirst-api/internal/gitopsCatalog"
	"github.com/kubefirst/kubefirst-api/pkg/k8s"
	"github.com/kubefirst/kubefirst-api/pkg/types"
	log "github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"
	"k8s.io/client-go/kubernetes"
)

const KUBEFIRST_CATALOG_SECRET_NAME = "kubefirst-catalog"

// GetGitopsCatalogApps
func GetGitopsCatalogApps(clientSet *kubernetes.Clientset) (types.GitopsCatalogApps, error) {
	catalogApps := types.GitopsCatalogApps{}

	kubefirstSecrets, err := k8s.ReadSecretV2(clientSet, "kubefirst", KUBEFIRST_CATALOG_SECRET_NAME)

	jsonString, err := MapToStructuredJSON(kubefirstSecrets)

	jsonData, err := json.Marshal(jsonString)
	if err != nil {
		return catalogApps, fmt.Errorf("error marshalling json: %s", err)
	}

	err = json.Unmarshal([]byte(jsonData), &catalogApps)
	if err != nil {
		return catalogApps, fmt.Errorf("unable to cast catalog: %s", err)
	}

	return catalogApps, nil
}

// GetGitopsCatalogAppsByCloudProvider
func GetGitopsCatalogAppsByCloudProvider(clientSet *kubernetes.Clientset, cloudProvider string, gitProvider string) (types.GitopsCatalogApps, error) {
	result, _ := GetGitopsCatalogApps(clientSet)

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

	catalogApps, _ := GetGitopsCatalogApps(clientSet)
	catalogApps.Apps = mpapps.Apps

	bytes, _ := json.Marshal(catalogApps)
	secretValuesMap, _ := ParseJSONToMap(string(bytes))

	err = k8s.UpdateSecretV2(clientSet, "kubefirst", KUBEFIRST_CATALOG_SECRET_NAME, secretValuesMap)

	if err != nil {
		return fmt.Errorf("error creating kubernetes secret: %s", err)
	}

	return nil
}
