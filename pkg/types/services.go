/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package types

// Service defines an individual cluster service
type Service struct {
	Name        string   `bson:"name" json:"name"`
	Default     bool     `bson:"default" json:"default"`
	Description string   `bson:"description" json:"description"`
	Image       string   `bson:"image" json:"image"`
	Links       []string `bson:"links" json:"links"`
	Status      string   `bson:"status" json:"status"`
	CreatedBy   string   `bson:"created_by" json:"created_by"`
}

// ClusterServiceList tracks services per cluster
type ClusterServiceList struct {
	ClusterName string    `bson:"cluster_name" json:"cluster_name"`
	Services    []Service `bson:"services" json:"services"`
}
