package constants

import "github.com/konstructio/kubefirst-api/pkg/types"

var cloudProviderDefaults = types.CloudProviderDefaults{
	Akamai:       types.CloudDefault{InstanceSize: "g6-standard-4", NodeCount: "4"},
	Aws:          types.CloudDefault{InstanceSize: "m5.large", NodeCount: "5"},
	Azure:        types.CloudDefault{InstanceSize: "Standard_D2_v4", NodeCount: "3"}, // @todo(sje): check these values are right
	Civo:         types.CloudDefault{InstanceSize: "g4s.kube.large", NodeCount: "4"},
	DigitalOcean: types.CloudDefault{InstanceSize: "s-4vcpu-8gb", NodeCount: "4"},
	Google:       types.CloudDefault{InstanceSize: "e2-medium", NodeCount: "2"},
	Vultr:        types.CloudDefault{InstanceSize: "vc2-4c-8gb", NodeCount: "4"},
	K3d:          types.CloudDefault{InstanceSize: "", NodeCount: "3"},
}

func GetCloudDefaults() types.CloudProviderDefaults {
	return cloudProviderDefaults
}
