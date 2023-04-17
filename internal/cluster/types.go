/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cluster

import "github.com/kubefirst/kubefirst-api/internal/types"

type ClusterEntry struct {
	Name       string
	Definition types.ClusterDefinition
}

type ClusterEntries struct {
	Clusters []ClusterEntry `yaml:"clusters"`
}
