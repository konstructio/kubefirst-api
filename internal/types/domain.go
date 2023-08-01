/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package types

// DomainListRequest
type DomainListRequest struct {
	CloudRegion      string           `json:"cloud_region"`
	AWSAuth          AWSAuth          `json:"aws_auth,omitempty"`
	CivoAuth         CivoAuth         `json:"civo_auth,omitempty"`
	DigitaloceanAuth DigitaloceanAuth `json:"do_auth,omitempty"`
	VultrAuth        VultrAuth        `json:"vultr_auth,omitempty"`
	CloudflareAuth   CloudflareAuth   `json:"cloudflare_auth,omitempty"`
}

// DomainListResponse
type DomainListResponse struct {
	Domains []string `json:"domains"`
}
