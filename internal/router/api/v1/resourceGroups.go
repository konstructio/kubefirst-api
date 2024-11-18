package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/konstructio/kubefirst-api/internal/azure"
	"github.com/konstructio/kubefirst-api/internal/types"
)

// Currently only needs to support google
func ListResourceGroups(c *gin.Context) {
	var resourceGroupsListRequest types.ResourceGroupsListRequest
	err := c.Bind(&resourceGroupsListRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	azureClient, err := azure.NewClient(
		resourceGroupsListRequest.AzureAuth.ClientID,
		resourceGroupsListRequest.AzureAuth.ClientSecret,
		resourceGroupsListRequest.AzureAuth.SubscriptionID,
		resourceGroupsListRequest.AzureAuth.TenantID,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	var resourceGroupsListResponse types.ResourceGroupsListResponse

	resourceGroups, err := azureClient.GetResourceGroups(context.Background())
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	resourceGroupsListResponse.ResourceGroups = resourceGroups

	c.JSON(http.StatusOK, resourceGroupsListResponse)
}
