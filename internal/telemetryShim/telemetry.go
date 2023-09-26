/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package telemetryShim

import (
	"os"
	"time"

	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/runtime/pkg/segment"
	log "github.com/sirupsen/logrus"
)

// Heartbeat
func Heartbeat(segmentClient *segment.SegmentClient) {
	TransmitClusterZero(true, segmentClient, segment.MetricKubefirstHeartbeat, "")
	HeartbeatWorkloadClusters()
	for range time.Tick(time.Minute * 20) {
		TransmitClusterZero(true, segmentClient, segment.MetricKubefirstHeartbeat, "")
		HeartbeatWorkloadClusters()
	}
}

// SetupTelemetry
func SetupTelemetry(cl pkgtypes.Cluster) (*segment.SegmentClient, error) {
	kubefirstVersion := os.Getenv("KUBEFIRST_VERSION")
	if kubefirstVersion == "" {
		kubefirstVersion = "development"
	}

	// Segment Client
	segmentClient := &segment.SegmentClient{
		CliVersion:        kubefirstVersion,
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

func SetupInitialTelemetry(clusterID string, clusterType string, installMethod string) (*segment.SegmentClient, error) {
	kubefirstVersion := os.Getenv("KUBEFIRST_VERSION")
	if kubefirstVersion == "" {
		kubefirstVersion = "development"
	}

	// Segment Client
	segmentClient := &segment.SegmentClient{
		CliVersion:        kubefirstVersion,
		ClusterID:         clusterID,
		ClusterType:       clusterType,
		KubefirstClient:   "api",
		KubefirstTeam:     os.Getenv("KUBEFIRST_TEAM"),
		KubefirstTeamInfo: os.Getenv("KUBEFIRST_TEAM_INFO"),
		InstallMethod:     installMethod,
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

func TransmitClusterZero(useTelemetry bool, segmentClient *segment.SegmentClient, metricName string, errorMessage string) {
	if useTelemetry {
		segmentMsg := segmentClient.SendCountClusterZeroMetric(metricName, errorMessage)
		if segmentMsg != "" {
			log.Info(segmentMsg)
		}
	}
}
