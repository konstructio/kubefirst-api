/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package types

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/civo/civogo"
	"github.com/digitalocean/godo"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	vultr "github.com/vultr/govultr/v3"
)

type InstanceSizesRequest struct {
	CloudRegion	string           	             `json:"cloud_region" binding:"required"`
	CivoAuth         pkgtypes.CivoAuth	       `json:"civo_auth,omitempty"`
	AWSAuth          pkgtypes.AWSAuth          `json:"aws_auth,omitempty"`
	DigitaloceanAuth pkgtypes.DigitaloceanAuth `json:"do_auth,omitempty"`
	VultrAuth        pkgtypes.VultrAuth        `json:"vultr_auth,omitempty"`
	GoogleAuth       pkgtypes.GoogleAuth       `json:"google_auth,omitempty"`
}

type CivoInstanceSizesResponse struct {
	InstanceSizes []civogo.InstanceSize `json:"instance_sizes"`
}

type AwsInstanceSizesResponse struct {
	InstanceSizes []types.InstanceTypeOffering `json:"instance_sizes"`
}

type DigitalOceanInstanceSizesResponse struct {
	InstanceSizes []*godo.AppInstanceSize `json:"instance_sizes"`
}

type VultrInstanceSizesResponse struct {
	InstanceSizes []vultr.Instance `json:"instance_sizes"`
}
