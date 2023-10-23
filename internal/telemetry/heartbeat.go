package telemetry

import (
	"fmt"
	"time"

	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	"github.com/segmentio/analytics-go"
	log "github.com/sirupsen/logrus"
)

func Heartbeat(segmentClient *telemetry.SegmentClient, dbClient *db.MongoDBClient) {

	fmt.Println("INSIDE HEARTBEAT")
	log.Info("LOG: INSIDE HEARTBEAT")

	telemetry.SendEvent(segmentClient, telemetry.KubefirstHeartbeat, "")
	HeartbeatWorkloadClusters(segmentClient, dbClient)

	for range time.Tick(time.Second * 30) {
		fmt.Println("INSIDE HEARTBEAT TICK TICK TICK")
		log.Info("LOG: INSIDE HEARTBEAT TICK TICK TICK")
		telemetry.SendEvent(segmentClient, telemetry.KubefirstHeartbeat, "")
		HeartbeatWorkloadClusters(segmentClient, dbClient)
	}
}

func HeartbeatWorkloadClusters(segmentClient *telemetry.SegmentClient, dbClient *db.MongoDBClient) error {

	clusters, _ := dbClient.GetClusters()
	fmt.Println("CLUSTERS: ", clusters)
	log.Info("LOG: CLUSTERS: ", clusters)

	for _, cluster := range clusters {
		if cluster.Status == constants.ClusterStatusProvisioned {
			fmt.Println("CLUSTER.STATUS: ", cluster.Status)
			log.Info("LOG: CLUSTER.STATUS:: ", cluster.Status)
			for _, workloadCluster := range cluster.WorkloadClusters {
				fmt.Println("CLUSTER.NAME: ", workloadCluster.ClusterName)
				fmt.Println("CLUSTER.STATUS: ", workloadCluster.Status)
				log.Info("LOG: CLUSTER.NAME:: ", workloadCluster.ClusterName)
				log.Info("LOG: CLUSTER.STATUS:: ", workloadCluster.Status)
				if workloadCluster.Status == constants.ClusterStatusProvisioned {

					workloadClient := &telemetry.SegmentClient{
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

					telemetry.SendEvent(workloadClient, telemetry.KubefirstHeartbeat, "")
					time.Sleep(time.Second * 3)
				}
			}
		}
	}

	return nil
}
