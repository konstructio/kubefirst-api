package azure

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

var defaultTags = map[string]*string{
	"ProvisionedBy": to.Ptr("kubefirst"),
}

type Keys struct {
	Key1 string
	Key2 string
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

func (c *Client) newSubscriptionClientFactory() (*armsubscriptions.ClientFactory, error) {
	client, err := armsubscriptions.NewClientFactory(c.cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create armsubscriptions client: %w", err)
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

func (c *Client) newVirtualMachineSizesClient() (*armcompute.VirtualMachineSizesClient, error) {
	client, err := armcompute.NewVirtualMachineSizesClient(c.subscriptionID, c.cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create virtualmachine client: %w", err)
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

func (c *Client) GetInstanceSizes(ctx context.Context, location string) ([]string, error) {
	client, err := c.newVirtualMachineSizesClient()
	if err != nil {
		return nil, err
	}

	var sizes []string

	pager := client.NewListPager(location, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list instance sizes: %w", err)
		}

		for _, v := range page.Value {
			sizes = append(sizes, *v.Name)
		}
	}

	return sizes, nil
}

func (c *Client) GetRegions(ctx context.Context) ([]string, error) {
	client, err := c.newSubscriptionClientFactory()
	if err != nil {
		return nil, err
	}

	pager := client.NewClient().NewListLocationsPager(c.subscriptionID, &armsubscriptions.ClientListLocationsOptions{
		IncludeExtendedLocations: to.Ptr(false),
	})

	var regions []string

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list regions: %w", err)
		}

		for _, v := range page.Value {
			regions = append(regions, *v.Name)
		}
	}

	return regions, nil
}

func (c *Client) GetStorageAccessKeys(ctx context.Context, resourceGroup, storageAccountName string) (*Keys, error) {
	client, err := c.newStorageClientFactory()
	if err != nil {
		return nil, err
	}

	keys, err := client.NewAccountsClient().ListKeys(ctx, resourceGroup, storageAccountName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve storage keys: %w", err)
	}

	// There should always be two keys set - this can be thought of as primary/secondary
	// so one in-use so other can be regenerated without losing access to the service
	s := make([]string, 0)
	for i, key := range keys.Keys {
		if k := key.Value; k != nil {
			s = append(s, *k)
		} else {
			return nil, fmt.Errorf("storage access key %d not set", i)
		}
	}

	return &Keys{
		Key1: s[0],
		Key2: s[1],
	}, nil
}

func (c *Client) ListResourceGroups(ctx context.Context) ([]*armresources.ResourceGroup, error) {
	client, err := c.newResourceClientFactory()
	if err != nil {
		return nil, err
	}

	pager := client.NewResourceGroupsClient().NewListPager(nil)

	var groups []*armresources.ResourceGroup

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list resource groups: %w", err)
		}

		groups = append(groups, page.Value...)
	}

	return groups, nil
}

func (c *Client) TestHostedZoneLivenessWildcard(ctx context.Context, domainName string) (bool, *string, error) {
	groups, err := c.ListResourceGroups(ctx)
	if err != nil {
		return false, nil, err
	}

	// Search through resource groups and return true for first match
	for _, resourceGroup := range groups {
		name := resourceGroup.Name
		hasDomain, err := c.TestHostedZoneLiveness(ctx, domainName, *name)
		if err != nil {
			return false, nil, err
		}

		if hasDomain {
			return true, name, nil
		}
	}

	return false, nil, nil
}

func (c *Client) TestHostedZoneLiveness(ctx context.Context, domainName, resourceGroup string) (bool, error) {
	client, err := c.newDNSClientFactory()
	if err != nil {
		return false, err
	}

	_, err = client.NewZonesClient().Get(ctx, resourceGroup, domainName, nil)
	if err != nil {
		//lint:ignore nilerr We cannot tell the difference between a network failure or a missing DNS zone
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
