/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package constants

import (
	"github.com/kubefirst/kubefirst-api/internal/types"
)

const (
	// The Namespace in which Kubefirst runs in-cluster
	KubefirstNamespace = "kubefirst"

	// The name of the Secret that holds authentication credentials
	KubefirstAuthSecretName = "kubefirst-secret"

	// Cluster statuses
	ClusterStatusDeleted      = "deleted"
	ClusterStatusDeleting     = "deleting"
	ClusterStatusError        = "error"
	ClusterStatusProvisioned  = "provisioned"
	ClusterStatusProvisioning = "provisioning"

	SilenceGetEnv = true
)

var cloudProviderDefaults = types.CloudProviderDefaults{
	Aws:          types.CloudDefault{InstanceSize: "m5.large", NodeCount: "6"},
	Civo:         types.CloudDefault{InstanceSize: "g4s.kube.large", NodeCount: "6"},
	DigitalOcean: types.CloudDefault{InstanceSize: "s-4vcpu-8gb", NodeCount: "4"},
	Google:       types.CloudDefault{InstanceSize: "e2-medium", NodeCount: "3"},
	Vultr:        types.CloudDefault{InstanceSize: "vc2-4c-8gb", NodeCount: "4"},
	K3d:          types.CloudDefault{InstanceSize: "", NodeCount: "3"},
}

func GetCloudDefaults() types.CloudProviderDefaults {
	return cloudProviderDefaults
}
