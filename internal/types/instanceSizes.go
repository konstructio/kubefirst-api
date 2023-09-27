/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package types

import (
	"github.com/civo/civogo"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
)

// DomainListRequest
type InstanceSizesRequest struct {
	CloudRegion	string           	`json:"cloud_region" binding:"required"`
	CivoAuth    pkgtypes.CivoAuth	`json:"civo_auth,omitempty"`
}

// DomainListResponse
type InstanceSizesResponse struct {
	InstanceSizes []civogo.InstanceSize `json:"instance_sizes"`
}

