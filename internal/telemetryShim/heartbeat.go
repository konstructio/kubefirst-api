/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package telemetryShim

import (
	"os"

	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/runtime/pkg/segment"
	log "github.com/sirupsen/logrus"
)

func HeartbeatWorkloadClusters() error {
	clusters, err := db.Client.GetClusters()

	if err != nil {
		log.Info("Clusters not found")
		return nil
	}

	kubefirstTeam := os.Getenv("KUBEFIRST_TEAM")
	if kubefirstTeam == "" {
		kubefirstTeam = "undefined"
	}

	for _, cluster := range clusters {
		if cluster.Status == constants.ClusterStatusProvisioned {
			for _, workloadCluster := range cluster.WorkloadClusters {
				if workloadCluster.Status == constants.ClusterStatusProvisioned {

					// Reusing telemetry function
					cluster = types.Cluster{}
					cluster.CloudProvider = workloadCluster.CloudProvider
					cluster.ClusterID = workloadCluster.ClusterID
					cluster.ClusterType = workloadCluster.ClusterType
					cluster.DomainName = workloadCluster.DomainName
					cluster.KubefirstTeam = kubefirstTeam

					// Setup telemetry
					segmentClient, err := SetupTelemetry(cluster)
					if err != nil {
						log.Warnf("Error sending workload cluster heartbeat %s", workloadCluster.ClusterID)
					}
					defer segmentClient.Client.Close()

					Transmit(true, segmentClient, segment.MetricKubefirstHeartbeat, "")
				}
			}
		}
	}

	return nil
}
