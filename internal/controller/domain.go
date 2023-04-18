/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"fmt"

	"github.com/kubefirst/runtime/pkg/civo"
	log "github.com/sirupsen/logrus"
)

// DomainLivenessTest
func (clctrl *ClusterController) DomainLivenessTest() error {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.DomainLivenessCheck {
		// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricDomainLivenessStarted, "")

		// domain id
		domainId, err := civo.GetDNSInfo(clctrl.DomainName, clctrl.CloudRegion)
		if err != nil {
			// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricDomainLivenessFailed, "domain liveness test failed")
			log.Info(err.Error())
		}

		log.Infof("domainId: %s", domainId)
		domainLiveness := civo.TestDomainLiveness(false, clctrl.DomainName, domainId, clctrl.CloudRegion)
		if !domainLiveness {
			return fmt.Errorf("failed to verify domain liveness for domain %s", clctrl.DomainName)
		}

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "domain_liveness_check", true)
		if err != nil {
			return err
		}

		// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricDomainLivenessCompleted, "")a
		log.Infof("domain %s verified", clctrl.DomainName)
	}

	return nil
}
