/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"context"
	"fmt"

	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	log "github.com/sirupsen/logrus"
)

// SetupMinioStorage
func SetupMinioStorage(kcfg *k8s.KubernetesClient, k1Dir string, gitProvider string) {
	ctx := context.Background()

	minioStopChannel := make(chan struct{}, 1)
	defer func() {
		close(minioStopChannel)
	}()
	k8s.OpenPortForwardPodWrapper(
		kcfg.Clientset,
		kcfg.RestConfig,
		"minio",
		"minio",
		9000,
		9000,
		minioStopChannel,
	)

	// Initialize minio client object.
	minioClient, err := minio.New(pkg.MinioPortForwardEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(pkg.MinioDefaultUsername, pkg.MinioDefaultPassword, ""),
		Secure: false,
		Region: pkg.MinioRegion,
	})

	if err != nil {
		log.Infof("Error creating Minio client: %s", err)
	}

	// define upload object
	objectName := fmt.Sprintf("terraform/%s/terraform.tfstate", gitProvider)
	filePath := k1Dir + fmt.Sprintf("/gitops/%s", objectName)
	contentType := "xl.meta"
	bucketName := "kubefirst-state-store"
	log.Infof("BucketName: %s", bucketName)

	// Upload the zip file with FPutObject
	info, err := minioClient.FPutObject(ctx, bucketName, objectName, filePath, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		log.Infof("Error uploading to Minio bucket: %s", err)
	}

	log.Printf("Successfully uploaded %s to bucket %s\n", objectName, info.Bucket)
}
