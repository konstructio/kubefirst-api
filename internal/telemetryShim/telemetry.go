/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package telemetryShim

import (
	"os"

	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/runtime/configs"
	"github.com/kubefirst/runtime/pkg/segment"
	log "github.com/sirupsen/logrus"
)

// SetupTelemetry
func SetupTelemetry(cl types.Cluster) (*segment.SegmentClient, error) {
	// Segment Client
	segmentClient := &segment.SegmentClient{
		CliVersion:        configs.K1Version,
		CloudProvider:     cl.CloudProvider,
		ClusterID:         cl.ClusterID,
		ClusterType:       cl.ClusterType,
		DomainName:        cl.DomainName,
		GitProvider:       cl.GitProvider,
		KubefirstClient:   "api",
		KubefirstTeam:     cl.KubefirstTeam,
		KubefirstTeamInfo: os.Getenv("KUBEFIRST_TEAM_INFO"),
	}
	segmentClient.SetupClient()

	return segmentClient, nil
}

// Transmit sends a metric via Segment
func Transmit(useTelemetry bool, segmentClient *segment.SegmentClient, metricName string, errorMessage string) {
	if useTelemetry {
		segmentMsg := segmentClient.SendCountMetric(metricName, errorMessage)
		if segmentMsg != "" {
			log.Info(segmentMsg)
		}
	}
}
