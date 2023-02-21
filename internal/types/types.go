/*
Copyright © 2023 Kubefirst <kubefirst.io>
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
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

// ClusterDefinition describes a Kubefirst management cluster
type ClusterDefinition struct {
	AdminEmail    string `json:"admin_email" binding:"required"`
	CloudProvider string `json:"cloud_provider" binding:"required,oneof=aws civo"`
	ClusterName   string `json:"cluster_name" binding:"required"`
	DomainName    string `json:"domain_name" binding:"required"`
	GitProvider   string `json:"git_provider" binding:"required,oneof=github gitlab"`
	GitHubOwner   string `json:"github_owner"`
	HostedZone    string `json:"hosted_zone"`
	Region        string `json:"region"`
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
