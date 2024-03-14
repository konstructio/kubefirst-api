/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package types

import (
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
)

// DomainListRequest
type DomainListRequest struct {
	CloudRegion      string                    `json:"cloud_region"`
	AkamaiAuth       pkgtypes.AkamaiAuth       `json:"akamai_auth,omitempty"`
	AWSAuth          pkgtypes.AWSAuth          `json:"aws_auth,omitempty"`
	CivoAuth         pkgtypes.CivoAuth         `json:"civo_auth,omitempty"`
	DigitaloceanAuth pkgtypes.DigitaloceanAuth `json:"do_auth,omitempty"`
	VultrAuth        pkgtypes.VultrAuth        `json:"vultr_auth,omitempty"`
	CloudflareAuth   pkgtypes.CloudflareAuth   `json:"cloudflare_auth,omitempty"`
	GoogleAuth       pkgtypes.GoogleAuth       `bson:"google_auth,omitempty" json:"google_auth,omitempty"`
}

// DomainListResponse
type DomainListResponse struct {
	Domains []string `json:"domains"`
}
