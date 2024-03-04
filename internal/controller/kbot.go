/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	pkg "github.com/kubefirst/kubefirst-api/pkg/utils"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	log "github.com/rs/zerolog/log"
)

// InitializeBot
func (clctrl *ClusterController) InitializeBot() error {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.KbotSetupCheck {

		clctrl.GitAuth.PrivateKey, clctrl.GitAuth.PublicKey, err = pkg.CreateSshKeyPair()
		if err != nil {
			log.Error().Msgf("error generating ssh keys: %s", err)
			telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.KbotSetupFailed, err.Error())
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
