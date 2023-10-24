package telemetry

import (
	"time"

	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
)

func Heartbeat(event telemetry.TelemetryEvent, dbClient *db.MongoDBClient) {

	telemetry.SendEvent(event, telemetry.KubefirstHeartbeat, "")
	HeartbeatWorkloadClusters(event, dbClient)

	for range time.Tick(time.Second * 300) {
		telemetry.SendEvent(event, telemetry.KubefirstHeartbeat, "")
		HeartbeatWorkloadClusters(event, dbClient)
	}
}

func HeartbeatWorkloadClusters(event telemetry.TelemetryEvent, dbClient *db.MongoDBClient) error {

	clusters, _ := dbClient.GetClusters()

	for _, cluster := range clusters {
		if cluster.Status == constants.ClusterStatusProvisioned {
			for _, workloadCluster := range cluster.WorkloadClusters {
				if workloadCluster.Status == constants.ClusterStatusProvisioned {

					telemetryEvent := telemetry.TelemetryEvent{
						CliVersion:        event.CliVersion,
						CloudProvider:     workloadCluster.CloudProvider,
						ClusterID:         workloadCluster.ClusterID,
						ClusterType:       workloadCluster.ClusterType,
						DomainName:        workloadCluster.DomainName,
						GitProvider:       event.GitProvider,
						InstallMethod:     event.InstallMethod,
						KubefirstClient:   event.KubefirstClient,
						KubefirstTeam:     event.KubefirstTeam,
						KubefirstTeamInfo: event.KubefirstTeamInfo,
						MachineID:         workloadCluster.DomainName,
						ErrorMessage:      "",
						UserId:            workloadCluster.DomainName,
						MetricName:        telemetry.KubefirstHeartbeat,
					}

					telemetry.SendEvent(telemetryEvent, telemetry.KubefirstHeartbeat, "")
				}
			}
		}
	}

	return nil
}
