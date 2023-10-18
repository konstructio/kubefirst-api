/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package types

import (
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
)

type InstanceSizesRequest struct {
	CloudRegion	string           	             `json:"cloud_region" binding:"required"`
	CloudZone	  string           	             `json:"cloud_zone,omitempty"`
	CivoAuth         pkgtypes.CivoAuth	       `json:"civo_auth,omitempty"`
	AWSAuth          pkgtypes.AWSAuth          `json:"aws_auth,omitempty"`
	DigitaloceanAuth pkgtypes.DigitaloceanAuth `json:"do_auth,omitempty"`
	VultrAuth        pkgtypes.VultrAuth        `json:"vultr_auth,omitempty"`
	GoogleAuth       pkgtypes.GoogleAuth       `json:"google_auth,omitempty"`
}

type InstanceSizesResponse struct {
	InstanceSizes []string `json:"instance_sizes"`
}
