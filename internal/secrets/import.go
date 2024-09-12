/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package secrets

import (
	"encoding/json"
	"fmt"

	"github.com/konstructio/kubefirst-api/internal/constants"
	"github.com/konstructio/kubefirst-api/internal/env"
	"github.com/konstructio/kubefirst-api/internal/k8s"
	pkgtypes "github.com/konstructio/kubefirst-api/pkg/types"
	"github.com/rs/zerolog/log"
)

func ImportClusterIfEmpty() (pkgtypes.Cluster, error) {
	// find the secret in mgmt cluster's kubefirst namespace and read import payload and clustername
	cluster := pkgtypes.Cluster{}
	env, _ := env.GetEnv(constants.SilenceGetEnv)

	if env.IsClusterZero == "true" {
		log.Info().Msg("IS_CLUSTER_ZERO is set to true, skipping import cluster logic.")
		return cluster, nil
	}

	kcfg, err := k8s.CreateKubeConfig(true, "")
	if err != nil {
		return cluster, fmt.Errorf("error creating kubeconfig: %w", err)
	}

	log.Info().Msg("reading secret kubefirst-initial-state to determine if import is needed")
	secData, err := k8s.ReadSecretV2Old(kcfg.Clientset, "kubefirst", "kubefirst-initial-state")
	if err != nil {
		log.Info().Msgf("error reading secret kubefirst-initial-state. %s", err)
		return cluster, fmt.Errorf("failed to read secret kubefirst-initial-state: %w", err)
	}

	jsonString, _ := MapToStructuredJSON(secData)

	jsonData, err := json.Marshal(jsonString)
	if err != nil {
		return cluster, fmt.Errorf("error marshalling json from secret data: %w", err)
	}

	err = json.Unmarshal(jsonData, &cluster)
	if err != nil {
		return cluster, fmt.Errorf("unable to cast unmarshalled JSON to cluster type: %w", err)
	}

	log.Info().Msgf("import cluster secret discovered for cluster %s", cluster.ClusterName)

	// if you find a record bail
	existingCluster, err := GetCluster(kcfg.Clientset, cluster.ClusterName)
	if err != nil {
		return existingCluster, fmt.Errorf("unable to find cluster: %w", err)
	}

	if existingCluster.ClusterID == "" {
		log.Info().Stack().Msgf("did not find preexisting record for cluster %s. importing record.", cluster.ClusterName)
		// Create if entry does not exist
		err = InsertCluster(kcfg.Clientset, cluster)
		if err != nil {
			return cluster, fmt.Errorf("error inserting cluster record %v into database: %w", cluster, err)
		}
		// log cluster
		log.Info().Msgf("inserted cluster record to db. adding default services. %s", cluster.ClusterName)

		return cluster, nil
	}

	log.Info().Msgf("cluster record for %s already exists - skipping", cluster.ClusterName)
	return pkgtypes.Cluster{}, nil
}
