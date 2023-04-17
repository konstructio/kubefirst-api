/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	internalssh "github.com/kubefirst/runtime/pkg/ssh"
)

// InitializeBot
func (clctrl *ClusterController) InitializeBot() error {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.KbotSetupCheck {
		sshPrivateKey, sshPublicKey, err := internalssh.CreateSshKeyPair()
		if err != nil {
			// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricKbotSetupFailed, err.Error())
			return err
		}

		clctrl.PublicKey = sshPublicKey
		clctrl.PrivateKey = sshPrivateKey

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "public_key", sshPublicKey)
		if err != nil {
			return err
		}
		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "private_key", sshPrivateKey)
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
