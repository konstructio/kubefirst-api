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
	Category    string                 `bson:"category" json:"category" yaml:"category"`
	ConfigKeys  []GitopsCatalogAppKeys `bson:"config_keys" json:"config_keys" yaml:"configKeys"`
	Description string                 `bson:"description" json:"description" yaml:"description"`
	DisplayName string                 `bson:"display_name" json:"display_name" yaml:"displayName"`
	ImageURL    string                 `bson:"image_url" json:"image_url" yaml:"imageUrl"`
	IsTemplate  bool                   `bson:"is_template" json:"is_template" yaml:"is_template"`
	Name        string                 `bson:"name" json:"name" yaml:"name"`
	SecretKeys  []GitopsCatalogAppKeys `bson:"secret_keys" json:"secret_keys" yaml:"secretKeys"`
}

// GitopsCatalogAppSecretKey describes a required secret value when creating a
// service based on a gitops catalog app
type GitopsCatalogAppKeys struct {
	Name  string `bson:"name" json:"name" yaml:"name"`
	Label string `bson:"label,omitempty" json:"label,omitempty" yaml:"label,omitempty"`
	Value string `bson:"value,omitempty" json:"value,omitempty" yaml:"value,omitempty"`
	Env   string `bson:"env,omitempty" json:"env,omitempty" yaml:"env,omitempty"`
}

// GitopsCatalogAppCreateRequest describes a request to create a service for a cluster
// based on a gitops catalog app
type GitopsCatalogAppCreateRequest struct {
	IsTemplate          bool                   `bson:"is_template" json:"is_template"`
	User                string                 `bson:"user" json:"user"`
	SecretKeys          []GitopsCatalogAppKeys `bson:"secret_keys,omitempty" json:"secret_keys,omitempty"`
	ConfigKeys          []GitopsCatalogAppKeys `bson:"config_keys,omitempty" json:"config_keys,omitempty"`
	WorkloadClusterName string                 `bson:"workload_cluster_name" json:"workload_cluster_name"`
}

type GitopsCatalogAppDeleteRequest struct {
	User                string `bson:"user" json:"user"`
	IsTemplate          bool   `bson:"is_template" json:"is_template"`
	WorkloadClusterName string `bson:"workload_cluster_name" json:"workload_cluster_name"`
}
