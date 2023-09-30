/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package types

// AWSProfileResponse is the response for the /aws/profiles route
type AWSDomainValidateResponse struct {
	Validated bool `json:"validated"`
}

// AWSProfileResponse is the response for the /aws/domain/validate route
type AWSProfilesResponse struct {
	Profiles []string `json:"profiles"`
}

// CivoDomainValidationRequest /civo/domain/validate required parameters
type CivoDomainValidationRequest struct {
	CloudRegion string `json:"cloud_region"`
}

// CivoDomainValidationResponse is the response for the /civo/domain/validate route
type CivoDomainValidationResponse struct {
	Validated bool `json:"validated"`
}

// DigitalOceanDomainValidationRequest /digitalocean/domain/validate required parameters
type DigitalOceanDomainValidationRequest struct {
	CloudRegion string `json:"cloud_region"`
}

// DigitalOceanDomainValidationResponse is the response for the /digitalocean/domain/validate route
type DigitalOceanDomainValidationResponse struct {
	Validated bool `json:"validated"`
}

// VultrDomainValidationRequest /vultr/domain/validate required parameters
type VultrDomainValidationRequest struct {
	CloudRegion string `json:"cloud_region"`
}

// VultrDomainValidationResponse is the response for the /vultr/domain/validate route
type VultrDomainValidationResponse struct {
	Validated bool `json:"validated"`
}

// GoogleDomainValidationRequest /google/domain/validate required parameters
type GoogleDomainValidationRequest struct {
	CloudRegion string `json:"cloud_region"`
}

// GoogleDomainValidationResponse is the response for the /google/domain/validate route
type GoogleDomainValidationResponse struct {
	Validated bool `json:"validated"`
}
