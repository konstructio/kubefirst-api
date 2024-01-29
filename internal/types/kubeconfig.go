package types

import (
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
)

type KubeconfigRequest struct {
	VCluster         bool                      `json:"vcluster,omitempty"`
	ClusterName      string                    `json:"cluster_name,required"`
	ManagClusterName string                    `json:"man_clust_name,omitempty"`
	CloudRegion      string                    `json:"cloud_region,omitempty"`
	CivoAuth         pkgtypes.CivoAuth         `json:"civo_auth,omitempty"`
	DigitaloceanAuth pkgtypes.DigitaloceanAuth `json:"do_auth,omitempty"`
	VultrAuth        pkgtypes.VultrAuth        `json:"vultr_auth,omitempty"`
}

type KubeconfigResponse struct {
	Config string `json:"config"`
}
