/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"fmt"

	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/runtime/pkg/civo"
	log "github.com/sirupsen/logrus"
)

// StateStoreCredentials
func (clctrl *ClusterController) StateStoreCredentials() error {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.StateStoreCredsCheck {
		creds, err := civo.GetAccessCredentials(clctrl.KubefirstStateStoreBucketName, clctrl.CloudRegion)
		if err != nil {
			log.Info(err.Error())
		}

		// Verify all credentials fields are present
		var civoCredsFailureMessage string
		switch {
		case creds.AccessKeyID == "":
			civoCredsFailureMessage = "when retrieving civo access credentials, AccessKeyID was empty - please retry your cluster creation"
		case creds.ID == "":
			civoCredsFailureMessage = "when retrieving civo access credentials, ID was empty - please retry your cluster creation"
		case creds.Name == "":
			civoCredsFailureMessage = "when retrieving civo access credentials, Name was empty - please retry your cluster creation"
		case creds.SecretAccessKeyID == "":
			civoCredsFailureMessage = "when retrieving civo access credentials, SecretAccessKeyID was empty - please retry your cluster creation"
		}
		if civoCredsFailureMessage != "" {
			// Creds failed to properly parse, so remove them
			err := civo.DeleteAccessCredentials(clctrl.KubefirstStateStoreBucketName, clctrl.CloudRegion)
			if err != nil {
				return err
			}

			// Return error
			return fmt.Errorf(civoCredsFailureMessage)
		}

		stateStoreData := types.StateStoreCredentials{
			AccessKeyID:     creds.AccessKeyID,
			SecretAccessKey: creds.SecretAccessKeyID,
			Name:            creds.Name,
			ID:              creds.ID,
		}
		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "state_store_credentials", stateStoreData)
		if err != nil {
			return err
		}

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "state_store_creds_check", true)
		if err != nil {
			return err
		}

		log.Infof("%s object storage credentials created and set", clctrl.CloudProvider)
	}

	return nil
}

// StateStoreCreate
func (clctrl *ClusterController) StateStoreCreate() error {
	cl, err := clctrl.MdbCl.GetCluster(clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.StateStoreCreateCheck {
		// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricStateStoreCreateStarted, "")

		accessKeyId := cl.StateStoreCredentials.AccessKeyID
		log.Infof("access key id %s", accessKeyId)

		bucket, err := civo.CreateStorageBucket(accessKeyId, clctrl.KubefirstStateStoreBucketName, clctrl.CloudRegion)
		if err != nil {
			// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricStateStoreCreateFailed, err.Error())
			log.Info(err.Error())
			return err
		}

		stateStoreData := types.StateStoreDetails{
			Name: bucket.Name,
			ID:   bucket.ID,
		}
		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "state_store_details", stateStoreData)
		if err != nil {
			return err
		}

		err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "state_store_create_check", true)
		if err != nil {
			return err
		}

		// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricStateStoreCreateCompleted, "")
		log.Infof("%s state store bucket created", clctrl.CloudProvider)
	}

	return nil
}
