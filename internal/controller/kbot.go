/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"os"

	"github.com/kubefirst/kubefirst-api/internal/telemetryShim"
	"github.com/kubefirst/runtime/pkg/segment"
	internalssh "github.com/kubefirst/runtime/pkg/ssh"
	log "github.com/sirupsen/logrus"
)

// InitializeBot
func (clctrl *ClusterController) InitializeBot() error {
	// Logging handler
	// Logs to stdout to maintain compatibility with event streaming
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "",
	})
	log.SetReportCaller(false)
	log.SetOutput(os.Stdout)

	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	// Telemetry handler
	segmentClient, err := telemetryShim.SetupTelemetry(cl)
	if err != nil {
		return err
	}
	defer segmentClient.Client.Close()

	if !cl.KbotSetupCheck {
		clctrl.GitAuth.PrivateKey, clctrl.GitAuth.PublicKey, err = internalssh.CreateSshKeyPair()
		if err != nil {
			log.Errorf("error generating ssh keys: %s", err)
			telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricKbotSetupFailed, err.Error())
			return err
		}

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "git_auth.public_key", clctrl.GitAuth.PublicKey)
		if err != nil {
			return err
		}
		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "git_auth.private_key", clctrl.GitAuth.PrivateKey)
		if err != nil {
			return err
		}

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "kbot_setup_check", true)
		if err != nil {
			return err
		}

	}

	return nil
}
