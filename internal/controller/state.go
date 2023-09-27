/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kubefirst/kubefirst-api/internal/telemetryShim"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/runtime/pkg/civo"
	"github.com/kubefirst/runtime/pkg/digitalocean"
	"github.com/kubefirst/runtime/pkg/segment"
	"github.com/kubefirst/runtime/pkg/vultr"
	log "github.com/sirupsen/logrus"
)

// StateStoreCredentials
func (clctrl *ClusterController) StateStoreCredentials() error {
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

	var stateStoreData pkgtypes.StateStoreCredentials

	if !cl.StateStoreCredsCheck {
		switch clctrl.CloudProvider {
		case "aws":
			kubefirstStateStoreBucket, err := clctrl.AwsClient.CreateBucket(clctrl.KubefirstStateStoreBucketName)
			if err != nil {
				return err
			}

			kubefirstArtifactsBucket, err := clctrl.AwsClient.CreateBucket(clctrl.KubefirstArtifactsBucketName)
			if err != nil {
				return err
			}

			stateStoreData = pkgtypes.StateStoreCredentials{
				AccessKeyID:     clctrl.AWSAuth.AccessKeyID,
				SecretAccessKey: clctrl.AWSAuth.SecretAccessKey,
				SessionToken:    clctrl.AWSAuth.SessionToken,
				Name:            clctrl.KubefirstStateStoreBucketName,
			}

			err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "state_store_details", pkgtypes.StateStoreDetails{
				AWSStateStoreBucket: strings.ReplaceAll(*kubefirstStateStoreBucket.Location, "/", ""),
				AWSArtifactsBucket:  strings.ReplaceAll(*kubefirstArtifactsBucket.Location, "/", ""),
				Hostname:            "s3.amazonaws.com",
				Name:                clctrl.KubefirstStateStoreBucketName,
			})
			if err != nil {
				return err
			}
		case "civo":
			civoConf := civo.CivoConfiguration{
				Client:  civo.NewCivo(cl.CivoAuth.Token, cl.CloudRegion),
				Context: context.Background(),
			}

			creds, err := civoConf.GetAccessCredentials(clctrl.KubefirstStateStoreBucketName, clctrl.CloudRegion)
			if err != nil {
				log.Error(err.Error())
			}

			stateStoreData = pkgtypes.StateStoreCredentials{
				AccessKeyID:     creds.AccessKeyID,
				SecretAccessKey: creds.SecretAccessKeyID,
				Name:            creds.Name,
				ID:              creds.ID,
			}
		case "digitalocean":
			digitaloceanConf := digitalocean.DigitaloceanConfiguration{
				Client:  digitalocean.NewDigitalocean(cl.DigitaloceanAuth.Token),
				Context: context.Background(),
			}

			creds := digitalocean.DigitaloceanSpacesCredentials{
				AccessKey:       cl.DigitaloceanAuth.SpacesKey,
				SecretAccessKey: cl.DigitaloceanAuth.SpacesSecret,
				Endpoint:        fmt.Sprintf("%s.digitaloceanspaces.com", "nyc3"),
			}
			err = digitaloceanConf.CreateSpaceBucket(creds, clctrl.KubefirstStateStoreBucketName)
			if err != nil {
				msg := fmt.Sprintf("error creating spaces bucket %s: %s", clctrl.KubefirstStateStoreBucketName, err)
				log.Error(msg)
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricStateStoreCreateFailed, msg)
				return fmt.Errorf(msg)
			}

			stateStoreData = pkgtypes.StateStoreCredentials{
				AccessKeyID:     creds.AccessKey,
				SecretAccessKey: creds.SecretAccessKey,
				Name:            clctrl.KubefirstStateStoreBucketName,
			}

			err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "state_store_details", pkgtypes.StateStoreDetails{
				Name:     clctrl.KubefirstStateStoreBucketName,
				Hostname: creds.Endpoint,
			})
			if err != nil {
				return err
			}

		case "google":
			// State is stored in a non s3 compliant gcs backend and thus the ADC provided will be used.

			// state store bucket created
			_, err := clctrl.GoogleClient.CreateBucket(clctrl.KubefirstStateStoreBucketName, []byte(clctrl.GoogleAuth.KeyFile))
			if err != nil {
				msg := fmt.Sprintf("error creating google bucket %s: %s", clctrl.KubefirstStateStoreBucketName, err)
				log.Error(msg)
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricStateStoreCreateFailed, msg)
				return fmt.Errorf(msg)
			}

		case "vultr":
			vultrConf := vultr.VultrConfiguration{
				Client:  vultr.NewVultr(cl.VultrAuth.Token),
				Context: context.Background(),
				Region:  cl.CloudRegion,
				// https://www.vultr.com/docs/vultr-object-storage/
				ObjectStorageRegion: "ewr",
			}

			objst, err := vultrConf.CreateObjectStorage(clctrl.KubefirstStateStoreBucketName)
			if err != nil {
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricStateStoreCreateFailed, err.Error())
				log.Error(err.Error())
				return err
			}
			err = vultrConf.CreateObjectStorageBucket(vultr.VultrBucketCredentials{
				AccessKey:       objst.S3AccessKey,
				SecretAccessKey: objst.S3SecretKey,
				Endpoint:        objst.S3Hostname,
			}, clctrl.KubefirstStateStoreBucketName)
			if err != nil {
				return fmt.Errorf("error creating vultr state storage bucket: %s", err)
			}

			stateStoreData = pkgtypes.StateStoreCredentials{
				AccessKeyID:     objst.S3AccessKey,
				SecretAccessKey: objst.S3SecretKey,
				Name:            objst.Label,
				ID:              objst.ID,
			}

			err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "state_store_details", pkgtypes.StateStoreDetails{
				Name:     objst.Label,
				ID:       objst.ID,
				Hostname: objst.S3Hostname,
			})
			if err != nil {
				return err
			}
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

	// Telemetry handler
	segmentClient, err := telemetryShim.SetupTelemetry(cl)
	if err != nil {
		return err
	}
	defer segmentClient.Client.Close()

	if !cl.StateStoreCreateCheck {
		switch clctrl.CloudProvider {
		case "civo":
			civoConf := civo.CivoConfiguration{
				Client:  civo.NewCivo(cl.CivoAuth.Token, cl.CloudRegion),
				Context: context.Background(),
			}

			telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricStateStoreCreateStarted, "")

			accessKeyId := cl.StateStoreCredentials.AccessKeyID
			log.Infof("access key id %s", accessKeyId)

			bucket, err := civoConf.CreateStorageBucket(accessKeyId, clctrl.KubefirstStateStoreBucketName, clctrl.CloudRegion)
			if err != nil {
				telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricStateStoreCreateFailed, err.Error())
				log.Error(err.Error())
				return err
			}

			stateStoreData := pkgtypes.StateStoreDetails{
				Name:     bucket.Name,
				ID:       bucket.ID,
				Hostname: bucket.BucketURL,
			}
			err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "state_store_details", stateStoreData)
			if err != nil {
				return err
			}

			err = clctrl.MdbCl.UpdateCluster(clctrl.ClusterName, "state_store_create_check", true)
			if err != nil {
				return err
			}

			telemetryShim.Transmit(clctrl.UseTelemetry, segmentClient, segment.MetricStateStoreCreateCompleted, "")
			log.Infof("%s state store bucket created", clctrl.CloudProvider)
		}
	}

	return nil
}
