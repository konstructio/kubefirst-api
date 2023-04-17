package k3d

import (
	"github.com/kubefirst/runtime/pkg/k3d"
	log "github.com/sirupsen/logrus"
)

// DeleteK3DCluster
func DeleteK3DCluster(clusterName string, k1Dir string, k3dClient string) error {
	log.Info("destroying k3d cluster")

	err := k3d.DeleteK3dCluster(clusterName, k1Dir, k3dClient)
	if err != nil {
		return err
	}

	log.Info("k3d resources terraform destroyed")

	return nil
}
