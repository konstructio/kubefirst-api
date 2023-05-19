/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package types

// MarketplaceApps lists all active marketplace app options
type MarketplaceApps struct {
	Apps []MarketplaceApp `bson:"apps" json:"apps" yaml:"apps"`
}

// MarketplaceApp describes a Kubefirst marketplace application
type MarketplaceApp struct {
	Name        string                    `bson:"name" json:"name" yaml:"name"`
	SecretKeys  []MarketplaceAppSecretKey `bson:"secret_keys" json:"secret_keys" yaml:"secretKeys"`
	ImageURL    string                    `bson:"image_url" json:"image_url" yaml:"imageUrl"`
	Description string                    `bson:"description" json:"description" yaml:"description"`
	Categories  []string                  `bson:"categories" json:"categories" yaml:"categories"`
}

// MarketplaceAppSecretKey describes a required secret value when creating a
// service based on a marketplace app
type MarketplaceAppSecretKey struct {
	Name  string `bson:"name" json:"name"`
	Label string `bson:"label,omitempty" json:"label,omitempty"`
	Value string `bson:"value,omitempty" json:"value,omitempty"`
}

// MarketplaceAppCreateRequest describes a request to create a service for a cluster
// based on a marketplace app
type MarketplaceAppCreateRequest struct {
	SecretKeys []MarketplaceAppSecretKey `bson:"secret_keys,omitempty" json:"secret_keys,omitempty"`
}
