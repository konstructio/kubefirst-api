/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package types

// RegionListRequest
type RegionListRequest struct {
	CloudRegion      string           `json:"cloud_region,omitempty"`
	AWSAuth          AWSAuth          `json:"aws_auth,omitempty"`
	CivoAuth         CivoAuth         `json:"civo_auth,omitempty"`
	DigitaloceanAuth DigitaloceanAuth `json:"do_auth,omitempty"`
	VultrAuth        VultrAuth        `json:"vultr_auth,omitempty"`
}

// RegionListResponse
type RegionListResponse struct {
	Regions []string `json:"regions"`
}
