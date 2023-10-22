package segment

import (
	"os"

	"github.com/denisbrodbeck/machineid"
	"github.com/kubefirst/kubefirst-api/pkg/metrics"
	"github.com/kubefirst/runtime/pkg/segment"

	"github.com/kubefirst/metrics-client/pkg/telemetry"
	"github.com/segmentio/analytics-go"
)

const (
	// SegmentIO constants
	// SegmentIOWriteKey The write key is the unique identifier for a source that tells Segment which source data comes
	// from, to which workspace the data belongs, and which destinations should receive the data.
	SegmentIOWriteKey        = "0gAYkX5RV3vt7s4pqCOOsDb6WHPLT30M"
	kubefirstClient   string = "api"
)

func InitClient() *telemetry.SegmentClient {

	machineID, _ := machineid.ID()

	c := telemetry.SegmentClient{
		TelemetryEvent: telemetry.TelemetryEvent{
			CliVersion:        os.Getenv("KUBEFIRST_VERSION"),
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
			ErrorMessage:      "",
			UserId:            machineID,
			MetricName:        metrics.KubefirstHeartbeat,
		},
		Client: analytics.New(segment.SegmentIOWriteKey),
	}

	return &c
}