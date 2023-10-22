package telemetry

import (
	"time"

	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/pkg/segment"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	"github.com/segmentio/analytics-go"
)

func Heartbeat(segmentClient *telemetry.SegmentClient, dbClient *db.MongoDBClient) {

	segClient := segment.InitClient()
	defer segClient.Client.Close()
	telemetry.SendEvent(segClient, telemetry.KubefirstHeartbeat, "")
	HeartbeatWorkloadClusters(segClient, dbClient)

	for range time.Tick(time.Second * 30) {
		telemetry.SendEvent(segClient, telemetry.KubefirstHeartbeat, "")
		HeartbeatWorkloadClusters(segClient, dbClient)
	}
}

func HeartbeatWorkloadClusters(segmentClient *telemetry.SegmentClient, dbClient *db.MongoDBClient) error {
	
	clusters, _ := dbClient.GetClusters()

	for _, cluster := range clusters {
		if cluster.Status == constants.ClusterStatusProvisioned {
			for _, workloadCluster := range cluster.WorkloadClusters {
				if workloadCluster.Status == constants.ClusterStatusProvisioned {

					workloadClient := telemetry.SegmentClient{
						TelemetryEvent: telemetry.TelemetryEvent{
							CliVersion:        segmentClient.TelemetryEvent.CliVersion,
							CloudProvider:     workloadCluster.CloudProvider,
							ClusterID:         workloadCluster.ClusterID,
							ClusterType:       workloadCluster.ClusterType,
							DomainName:        workloadCluster.DomainName,
							GitProvider:       segmentClient.TelemetryEvent.GitProvider,
							InstallMethod:     segmentClient.TelemetryEvent.InstallMethod,
							KubefirstClient:   segmentClient.TelemetryEvent.KubefirstClient,
							KubefirstTeam:     segmentClient.TelemetryEvent.KubefirstTeam,
							KubefirstTeamInfo: segmentClient.TelemetryEvent.KubefirstTeamInfo,
							MachineID:         segmentClient.TelemetryEvent.MachineID,
							ErrorMessage:      "",
							UserId:            segmentClient.TelemetryEvent.MachineID,
							MetricName:        telemetry.KubefirstHeartbeat,
						},
						Client: analytics.New(telemetry.SegmentIOWriteKey),
					}
					defer workloadClient.Client.Close()

					telemetry.SendEvent(&workloadClient, telemetry.KubefirstHeartbeat, "")
				}
			}
		}
	}

	return nil
}
