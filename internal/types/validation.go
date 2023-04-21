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
