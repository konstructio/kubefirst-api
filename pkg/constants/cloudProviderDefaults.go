package constants

import "github.com/kubefirst/kubefirst-api/pkg/types"

var cloudProviderDefaults = types.CloudProviderDefaults{
	Aws:          types.CloudDefault{InstanceSize: "m5.large", NodeCount: 6},
	Civo:         types.CloudDefault{InstanceSize: "g4s.kube.large", NodeCount: 6},
	DigitalOcean: types.CloudDefault{InstanceSize: "s-4vcpu-8gb", NodeCount: 4},
	Google:       types.CloudDefault{InstanceSize: "e2-medium", NodeCount: 2},
	Vultr:        types.CloudDefault{InstanceSize: "vc2-4c-8gb", NodeCount: 4},
	K3d:          types.CloudDefault{InstanceSize: "", NodeCount: 3},
}

func GetCloudDefaults() types.CloudProviderDefaults {
	return cloudProviderDefaults
}
