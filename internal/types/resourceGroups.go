package types

import pkgtypes "github.com/konstructio/kubefirst-api/pkg/types"

type ResourceGroupsListRequest struct {
	AzureAuth pkgtypes.AzureAuth `bson:"azure_auth,omitempty" json:"azure_auth,omitempty"`
}

type ResourceGroupsListResponse struct {
	ResourceGroups []string `json:"resource_groups"`
}
