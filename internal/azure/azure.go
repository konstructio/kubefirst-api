package azure

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

var defaultTags = map[string]*string{
	"ProvisionedBy": to.Ptr("kubefirst"),
}

type Client struct {
	cred           *azidentity.DefaultAzureCredential
	subscriptionID string
}

func (c *Client) newDNSClientFactory() (*armdns.ClientFactory, error) {
	client, err := armdns.NewClientFactory(c.subscriptionID, c.cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create armdns client: %w", err)
	}
	return client, nil
}

func (c *Client) newResourceClientFactory() (*armresources.ClientFactory, error) {
	client, err := armresources.NewClientFactory(c.subscriptionID, c.cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create armresources client: %w", err)
	}
	return client, nil
}

func (c *Client) newStorageClientFactory() (*armstorage.ClientFactory, error) {
	client, err := armstorage.NewClientFactory(c.subscriptionID, c.cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create armstorage client: %w", err)
	}
	return client, nil
}

func (c *Client) CreateBlobContainer(ctx context.Context, storageAccountName, containerName string) (*azblob.CreateContainerResponse, error) {
	client, err := azblob.NewClient(fmt.Sprintf("https://%s.blob.core.windows.net", storageAccountName), c.cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create azblob client: %w", err)
	}

	resp, err := client.CreateContainer(ctx, containerName, &azblob.CreateContainerOptions{
		Metadata: defaultTags,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	return &resp, nil
}

func (c *Client) CreateResourceGroup(ctx context.Context, name, location string) (*armresources.ResourceGroup, error) {
	client, err := c.newResourceClientFactory()
	if err != nil {
		return nil, err
	}

	parameters := armresources.ResourceGroup{
		Location: to.Ptr(location),
		Tags:     defaultTags,
	}

	resp, err := client.NewResourceGroupsClient().CreateOrUpdate(ctx, name, parameters, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create azure resource group: %w", err)
	}

	return &resp.ResourceGroup, nil
}

func (c *Client) CreateStorageAccount(ctx context.Context, location, resourceGroup, storageAccountName string) (*armstorage.Account, error) {
	client, err := c.newStorageClientFactory()
	if err != nil {
		return nil, err
	}

	params := armstorage.AccountCreateParameters{
		Kind:     to.Ptr(armstorage.KindStorageV2),
		Location: to.Ptr(location),
		SKU: &armstorage.SKU{
			Name: to.Ptr(armstorage.SKUNameStandardGRS),
		},
		Properties: &armstorage.AccountPropertiesCreateParameters{
			AccessTier:            to.Ptr(armstorage.AccessTierCool),
			AllowBlobPublicAccess: to.Ptr(false),
			Encryption: &armstorage.Encryption{
				KeySource: to.Ptr(armstorage.KeySourceMicrosoftStorage),
				Services: &armstorage.EncryptionServices{
					// We're only using blob storage here, so the other types aren't set
					Blob: &armstorage.EncryptionService{
						Enabled: to.Ptr(true),
						KeyType: to.Ptr(armstorage.KeyTypeAccount),
					},
				},
			},
			MinimumTLSVersion: to.Ptr(armstorage.MinimumTLSVersionTLS12),
		},
		Tags: defaultTags,
	}

	poller, err := client.NewAccountsClient().BeginCreate(
		ctx,
		resourceGroup,
		storageAccountName,
		params,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("storage account creation request failed: %w", err)
	}

	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage account: %w", err)
	}

	return &resp.Account, nil
}

func (c *Client) TestHostedZoneLiveness(ctx context.Context, domainName, resourceGroup string) (bool, error) {
	client, err := c.newDNSClientFactory()
	if err != nil {
		return false, err
	}

	_, err = client.NewZonesClient().Get(ctx, resourceGroup, domainName, nil)
	if err != nil {
		// We cannot tell the difference between a network failure or a missing DNS zone
		return false, nil
	}

	return true, nil
}

func NewClient(clientID, clientSecret, subscriptionID, tenantID string) (*Client, error) {
	// I don't particularly like this, but there doesn't seem to be any other way
	// of achieving this with the SDK. If someone knows a way, please open a PR
	//
	// @link https://learn.microsoft.com/en-us/azure/developer/go/azure-sdk-authentication-service-principal?tabs=azure-cli
	os.Setenv("AZURE_CLIENT_ID", clientID)
	os.Setenv("AZURE_CLIENT_SECRET", clientSecret)
	os.Setenv("AZURE_TENANT_ID", tenantID)

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create default azure credential: %w", err)
	}

	return &Client{
		cred:           cred,
		subscriptionID: subscriptionID,
	}, nil
}
