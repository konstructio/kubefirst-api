package types

import "fmt"

// AzureAuth holds necessary auth credentials for interacting with azure
type AzureAuth struct {
	ClientID       string `bson:"client_id" json:"client_id"`
	ClientSecret   string `bson:"client_secret" json:"client_secret"`
	TenantID       string `bson:"tenant_id" json:"tenant_id"`
	SubscriptionID string `bson:"subscription_id" json:"subscription_id"`
}

func (auth *AzureAuth) ValidateAuthCredentials() error {
	if auth.ClientID == "" ||
		auth.ClientSecret == "" ||
		auth.SubscriptionID == "" ||
		auth.TenantID == "" {
		return fmt.Errorf("missing authentication credentials in request, please check and try again")
	}

	return nil
}
