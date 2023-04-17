/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"github.com/kubefirst/runtime/pkg/k3d"
	log "github.com/sirupsen/logrus"
)

// DownloadTools
// This obviously doesn't work in an api-based environment.
// It's included for testing and development.
func (clctrl *ClusterController) DownloadTools(gitProvider string, gitOwner string, toolsDir string) error {
	log.Info("installing kubefirst dependencies")

	err := k3d.DownloadTools(gitProvider, gitOwner, toolsDir)
	if err != nil {
		return err
	}
	log.Info("download dependencies `$HOME/.k1/tools` complete")

	return nil
}
