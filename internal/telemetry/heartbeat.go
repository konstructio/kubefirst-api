package telemetry

import (
	"time"

	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/pkg/metrics"
	"github.com/kubefirst/kubefirst-api/pkg/segment"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	"github.com/segmentio/analytics-go"
)

func Heartbeat(segmentClient *telemetry.SegmentClient, dbClient *db.MongoDBClient) {
	telemetry.SendCountMetric(segmentClient, metrics.KubefirstHeartbeat, "")
	HeartbeatWorkloadClusters(segmentClient, dbClient)

	for range time.Tick(time.Second * 30) {
		telemetry.SendCountMetric(segmentClient, metrics.KubefirstHeartbeat, "")
		HeartbeatWorkloadClusters(segmentClient, dbClient)
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
							MetricName:        metrics.KubefirstHeartbeat,
						},
						Client: analytics.New(segment.SegmentIOWriteKey),
					}
					defer workloadClient.Client.Close()

					telemetry.SendCountMetric(&workloadClient, metrics.KubefirstHeartbeat, "")
				}
			}
		}
	}

	return nil
}