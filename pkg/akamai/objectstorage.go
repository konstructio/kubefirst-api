package akamai

import (
	"context"

	"github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/linode/linodego"
)

// CreateObjectStorageBucketAndKeys creates object store and access credentials
func (c *AkamaiConfiguration) CreateObjectStorageBucketAndKeys(clusterName string) (AkamaiBucketAndKeysConfiguration, error) {
	// todo get rid of hardcode default
	DEFAULT_CLUSTER := "us-east-1"
	bucket, err := c.Client.CreateObjectStorageBucket(context.TODO(), linodego.ObjectStorageBucketCreateOptions{
		Cluster: DEFAULT_CLUSTER,
		Label:   clusterName,
	})
	if err != nil {
		return AkamaiBucketAndKeysConfiguration{}, err
	}

	creds, err := c.Client.CreateObjectStorageKey(context.TODO(), linodego.ObjectStorageKeyCreateOptions{
		Label: clusterName,
		BucketAccess: &[]linodego.ObjectStorageKeyBucketAccess{
			{
				BucketName:  clusterName,
				Cluster:     DEFAULT_CLUSTER,
				Permissions: "read_write",
			},
		},
	})

	// todo add validation
	stateStoreData := types.StateStoreDetails{
		Name:     bucket.Label,
		ID:       bucket.Hostname,
		Hostname: bucket.Hostname,
	}

	stateStoreCredentialsData := types.StateStoreCredentials{
		AccessKeyID:     creds.AccessKey,
		SecretAccessKey: creds.SecretKey,
		Name:            bucket.Label,
	}

	return AkamaiBucketAndKeysConfiguration{stateStoreData, stateStoreCredentialsData}, nil
}
