/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package objectStorage

import (
	"context"
	"fmt"
	"io"
	"os"

	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	log "github.com/rs/zerolog/log"
)

// PutBucketObject
func PutBucketObject(cr *pkgtypes.StateStoreCredentials, d *pkgtypes.StateStoreDetails, obj *pkgtypes.PushBucketObject) error {
	ctx := context.Background()

	// Initialize minio client object.
	minioClient, err := minio.New(d.Hostname, &minio.Options{
		Creds:  credentials.NewStaticV4(cr.AccessKeyID, cr.SecretAccessKey, ""),
		Secure: true,
	})
	if err != nil {
		return fmt.Errorf("error initializing minio client: %s", err)
	}

	object, err := os.Open(obj.LocalFilePath)
	if err != nil {
		return err
	}
	defer object.Close()

	objectStat, err := object.Stat()
	if err != nil {
		return err
	}

	n, err := minioClient.PutObject(ctx, d.Name, obj.RemoteFilePath, object, objectStat.Size(), minio.PutObjectOptions{ContentType: obj.ContentType})
	if err != nil {
		return err
	}
	log.Info().Msgf("uploaded %s of size: %d successfully", obj.LocalFilePath, n.Size)

	return nil
}

// PutClusterObject exports a cluster definition as json and places it in the target object storage bucket
func PutClusterObject(cr *pkgtypes.StateStoreCredentials, d *pkgtypes.StateStoreDetails, obj *pkgtypes.PushBucketObject) error {
	ctx := context.Background()

	// Initialize minio client
	minioClient, err := minio.New(d.Hostname, &minio.Options{
		Creds:  credentials.NewStaticV4(cr.AccessKeyID, cr.SecretAccessKey, cr.SessionToken),
		Secure: true,
	})
	if err != nil {
		return fmt.Errorf("error initializing minio client: %s", err)
	}

	// Reference for cluster object output file
	object, err := os.Open(obj.LocalFilePath)
	if err != nil {
		return fmt.Errorf("error during object local copy file lookup: %s", err)
	}
	defer object.Close()

	objectStat, err := object.Stat()
	if err != nil {
		return fmt.Errorf("error during object stat: %s", err)
	}

	// Put
	_, err = minioClient.PutObject(
		ctx,
		d.Name,
		obj.RemoteFilePath,
		object,
		objectStat.Size(),
		minio.PutObjectOptions{ContentType: obj.ContentType},
	)
	if err != nil {
		return fmt.Errorf("error during object put: %s", err)
	}
	log.Info().Msgf("uploaded cluster object %s to state store bucket %s successfully", obj.LocalFilePath, d.Name)

	return nil
}

// GetClusterObject imports a cluster definition as json
func GetClusterObject(cr *pkgtypes.StateStoreCredentials, d *pkgtypes.StateStoreDetails, localFilePath string, remoteFilePath string, secure bool) error {
	ctx := context.Background()

	// Initialize minio client
	minioClient, err := minio.New(d.Hostname, &minio.Options{
		Creds:  credentials.NewStaticV4(cr.AccessKeyID, cr.SecretAccessKey, cr.SessionToken),
		Secure: secure,
	})
	if err != nil {
		return fmt.Errorf("error initializing minio client: %s", err)
	}

	_, err = minioClient.BucketExists(ctx, d.Name)
	if err != nil {
		log.Info().Msg(err.Error())
		return err
	}

	// Get object from bucket
	reader, err := minioClient.GetObject(ctx, d.Name, remoteFilePath, minio.GetObjectOptions{})
	if err != nil {
		log.Info().Msg(err.Error())
		return fmt.Errorf("error retrieving cluster object from bucket: %s", err)
	}
	defer reader.Close()

	// Write object to local file
	localFile, err := os.Create(localFilePath)
	if err != nil {
		return fmt.Errorf("error during object local copy file create: %s", err)
	}
	defer localFile.Close()

	stat, err := reader.Stat()
	if err != nil {
		return fmt.Errorf("error during object stat: %s", err)
	}
	if _, err := io.CopyN(localFile, reader, stat.Size); err != nil {
		return fmt.Errorf("error during object copy: %s", err)
	}

	return nil
}
