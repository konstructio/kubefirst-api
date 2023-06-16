/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package types

// GitopsCatalogApps lists all active gitops catalog app options
type GitopsCatalogApps struct {
	Name string             `bson:"name" json:"name" yaml:"name"`
	Apps []GitopsCatalogApp `bson:"apps" json:"apps" yaml:"apps"`
}

// GitopsCatalogApp describes a Kubefirst gitops catalog application
type GitopsCatalogApp struct {
	Name        string                      `bson:"name" json:"name" yaml:"name"`
	DisplayName string                      `bson:"display_name" json:"display_name" yaml:"displayName"`
	SecretKeys  []GitopsCatalogAppSecretKey `bson:"secret_keys" json:"secret_keys" yaml:"secretKeys"`
	ImageURL    string                      `bson:"image_url" json:"image_url" yaml:"imageUrl"`
	Description string                      `bson:"description" json:"description" yaml:"description"`
	Categories  []string                    `bson:"categories" json:"categories" yaml:"categories"`
}

// GitopsCatalogAppSecretKey describes a required secret value when creating a
// service based on a gitops catalog app
type GitopsCatalogAppSecretKey struct {
	Name  string `bson:"name" json:"name"`
	Label string `bson:"label,omitempty" json:"label,omitempty"`
	Value string `bson:"value,omitempty" json:"value,omitempty"`
}

// GitopsCatalogAppCreateRequest describes a request to create a service for a cluster
// based on a gitops catalog app
type GitopsCatalogAppCreateRequest struct {
	SecretKeys []GitopsCatalogAppSecretKey `bson:"secret_keys,omitempty" json:"secret_keys,omitempty"`
}
