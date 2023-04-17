/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"os"

	"github.com/kubefirst/runtime/pkg/segment"
	log "github.com/sirupsen/logrus"
)

func (clctrl *ClusterController) SetupTelemetry() (*segment.SegmentClient, error) {
	// Segment Client
	segmentClient := &segment.SegmentClient{
		// CliVersion:        clctrl.Version,
		CloudProvider:     clctrl.CloudProvider,
		ClusterID:         clctrl.ClusterID,
		ClusterType:       clctrl.ClusterType,
		DomainName:        clctrl.DomainName,
		GitProvider:       clctrl.GitProvider,
		KubefirstTeam:     clctrl.KubefirstTeam,
		KubefirstTeamInfo: os.Getenv("KUBEFIRST_TEAM_INFO"),
	}
	segmentClient.SetupClient()

	// This defer func likely needs to get passed in and referenced anywhere telemetry is used
	defer func(c segment.SegmentClient) {
		err := c.Client.Close()
		if err != nil {
			log.Infof("error closing segment client %s", err.Error())
		}
	}(*segmentClient)

	return segmentClient, nil
}
