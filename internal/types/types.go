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

// CivoDomainValidationResponse is the response for the /civo/domain/validate route
type CivoDomainValidationResponse struct {
	Validated bool `json:"validated"`
}

// ClusterDefinition describes a Kubefirst cluster
type ClusterDefinition struct {
	AdminEmail    string `json:"admin_email" binding:"required"`
	CloudProvider string `json:"cloud_provider" binding:"required,oneof=aws civo digitalocean k3d vultr"`
	CloudRegion   string `json:"cloud_region" binding:"required"`
	ClusterName   string `json:"cluster_name" binding:"required"`
	DomainName    string `json:"domain_name" binding:"required"`
	GitProvider   string `json:"git_provider" binding:"required,oneof=github gitlab"`
	GitOwner      string `json:"git_owner" binding:"required"`
	GitToken      string `json:"git_token" binding:"required"`
	HostedZone    string `json:"hosted_zone"`
	Type          string `json:"type" binding:"required,oneof=mgmt workload"`
}

// JSONFailureResponse describes a failure message returned by the API
type JSONFailureResponse struct {
	Message string `json:"error" example:"err"`
}

// JSONHealthResponse describes a message returned by the API health endpoint
type JSONHealthResponse struct {
	Status string `json:"status" example:"healthy"`
}

// JSONSuccessResponse describes a success message returned by the API
type JSONSuccessResponse struct {
	Message string `json:"message" example:"success"`
}
