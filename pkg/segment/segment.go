package segment

import (
	"os"

	"github.com/denisbrodbeck/machineid"
	"github.com/kubefirst/metrics-client/pkg/telemetry"

	log "github.com/rs/zerolog/log"
	"github.com/segmentio/analytics-go"
)

const (
	kubefirstClient string = "api"
)

func InitClient() *telemetry.SegmentClient {

	machineID, err := machineid.ID()
	if err != nil {
		log.Info().Msg("machine id FAILED")
	}
	sc := analytics.New(telemetry.SegmentIOWriteKey)

	kubefirstVersion := os.Getenv("KUBEFIRST_VERSION")
	if kubefirstVersion == "" {
		kubefirstVersion = "development"
	}

	c := telemetry.SegmentClient{
		TelemetryEvent: telemetry.TelemetryEvent{
			CliVersion:        kubefirstVersion,
			CloudProvider:     os.Getenv("CLOUD_PROVIDER"),
			ClusterID:         os.Getenv("CLUSTER_ID"),
			ClusterType:       os.Getenv("CLUSTER_TYPE"),
			DomainName:        os.Getenv("DOMAIN_NAME"),
			GitProvider:       os.Getenv("GIT_PROVIDER"),
			InstallMethod:     os.Getenv("INSTALL_METHOD"),
			KubefirstClient:   kubefirstClient,
			KubefirstTeam:     os.Getenv("KUBEFIRST_TEAM"),
			KubefirstTeamInfo: os.Getenv("KUBEFIRST_TEAM_INFO"),
			MachineID:         machineID,
			ParentClusterId:   os.Getenv("PARENT_CLUSTER_ID"),
			ErrorMessage:      "",
			UserId:            machineID,
			MetricName:        telemetry.ClusterInstallCompleted,
		},
		Client: sc,
	}

	return &c
}
