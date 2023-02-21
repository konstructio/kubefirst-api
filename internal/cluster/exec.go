/*
Copyright © 2023 Kubefirst <kubefirst.io>
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cluster

import (
	"errors"
	"os"

	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/kubefirst-api/internal/utils"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const (
	clusterManagementFilePath string = "clusters.yaml"
)

// CreateCluster adds a managed cluster to the config file
func CreateCluster(clusterName string, definition types.ClusterDefinition) error {
	yamlSample := ClusterEntry{
		Name:       clusterName,
		Definition: definition,
	}

	err := yamlSample.Save()
	if err != nil {
		return err
	}
	return nil
}

// GetCluster returns a single configured cluster
func GetCluster(clusterName string) (*ClusterEntry, error) {
	cl := ClusterEntries{}
	ro, err := cl.ReadOne(clusterName)
	if err != nil {
		return &ClusterEntry{}, err
	}
	return &ro, nil
}

// GetClusters returns all configured clusters
func GetClusters() (*[]ClusterEntry, error) {
	cl := ClusterEntries{}
	ro, err := cl.ReadAll()
	if err != nil {
		return &[]ClusterEntry{}, err
	}
	return &ro, nil
}

// DeleteOne deletes a configured cluster if it exists
func (c *ClusterEntries) DeleteOne(clusterName string) error {
	// Determine if file already exists
	_, err := os.Open(clusterManagementFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return err
		}
	}

	// Read file
	buffer, err := os.ReadFile(clusterManagementFilePath)
	if err != nil {
		log.Errorf("error reading file %s: %s", clusterManagementFilePath, err.Error())
	}

	// Parse existing entries
	existingEntries := ClusterEntries{}
	err = yaml.Unmarshal(buffer, &existingEntries)

	// Return any matches
	discoveredClusters := make([]ClusterEntry, 0)
	for _, cl := range existingEntries.Clusters {
		discoveredClusters = append(discoveredClusters, cl)
	}

	// Determine if an index matches
	var indexToRemove int = -1
	for i, cl := range discoveredClusters {
		if cl.Name == clusterName {
			indexToRemove = i
		}
	}

	switch {
	// Remove target cluster if an index was retrieved
	case indexToRemove != -1:
		alteredClusters := utils.RemoveFromSlice(discoveredClusters, indexToRemove)
		alteredClusterEntries := ClusterEntries{
			Clusters: alteredClusters,
		}
		// Rewrite config
		m, err := yaml.Marshal(&alteredClusterEntries)
		err = os.WriteFile(clusterManagementFilePath, m, 0600)
		if err != nil {
			log.Error(err)
		}
	// Otherwise, return an error
	case indexToRemove == -1:
		return errors.New("cluster does not exist")
	}
	return nil
}

// ReadOne returns a single cluster definition if it exists
func (c *ClusterEntries) ReadOne(clusterName string) (ClusterEntry, error) {
	// Determine if file already exists
	_, err := os.Open(clusterManagementFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ClusterEntry{}, err
		}
	}

	// Read file
	buffer, err := os.ReadFile(clusterManagementFilePath)
	if err != nil {
		log.Errorf("error reading file %s: %s", clusterManagementFilePath, err.Error())
	}

	// Parse existing entries
	existingEntries := ClusterEntries{}
	err = yaml.Unmarshal(buffer, &existingEntries)

	// Return any matches
	for _, cl := range existingEntries.Clusters {
		if cl.Name == clusterName {
			return cl, nil
		}
	}
	return ClusterEntry{}, errors.New("cluster not found")
}

// ReadAll all existing cluster definitions
func (c *ClusterEntries) ReadAll() ([]ClusterEntry, error) {
	// Determine if file already exists
	_, err := os.Open(clusterManagementFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ClusterEntry{}, err
		}
	}

	// Read file
	buffer, err := os.ReadFile(clusterManagementFilePath)
	if err != nil {
		log.Errorf("error reading file %s: %s", clusterManagementFilePath, err.Error())
	}

	// Parse existing entries
	existingEntries := ClusterEntries{}
	err = yaml.Unmarshal(buffer, &existingEntries)

	// Return all clusters
	discoveredClusters := make([]ClusterEntry, 0)
	for _, cl := range existingEntries.Clusters {
		discoveredClusters = append(discoveredClusters, cl)
	}
	return discoveredClusters, nil
}

// Save writes a ClusterEntry to the list of managed clusters
// todo: cluster names should be unique, so this should traverse keys to match
// on existing cluster names before saving or return an error if a cluster already
// exists
func (c *ClusterEntry) Save() error {
	// Determine if file already exists
	_, err := os.Open(clusterManagementFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Infof("file %s does not exist and will be created", clusterManagementFilePath)
			yfile, err := os.Create(clusterManagementFilePath)
			if err != nil {
				return err
			}
			enc := yaml.NewEncoder(yfile)
			err = enc.Encode(map[string][]string{"clusters": {}})
		}
	}

	// Read file
	buffer, err := os.ReadFile(clusterManagementFilePath)
	if err != nil {
		return err
	}

	// Parse existing entries
	existingEntries := ClusterEntries{}
	err = yaml.Unmarshal(buffer, &existingEntries)

	// Make sure the cluster doesn't already exist
	var foundExistingClusterEntry = false
	for _, cl := range existingEntries.Clusters {
		if cl.Name == c.Name {
			foundExistingClusterEntry = true
		}
	}

	// Add cluster config to config file if it does not exist
	switch foundExistingClusterEntry {
	case true:
		return errors.New("cluster already exists")
	case false:
		// Append new entry if it doesn't exist
		existingEntries.Clusters = append(existingEntries.Clusters, *c)
		m, err := yaml.Marshal(&existingEntries)
		err = os.WriteFile(clusterManagementFilePath, m, 0600)
		return err
	}
	return err
}
