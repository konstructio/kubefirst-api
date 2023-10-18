/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package types

import (
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
)

type ZonesListRequest struct {
	CloudRegion	string           	  `json:"cloud_region" binding:"required"`
	GoogleAuth  pkgtypes.GoogleAuth `json:"google_auth" binding:"required"`
}

type ZonesListResponse struct {
	Zones []string `json:"zones"`
}
