/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package db

import (
	"fmt"

	"github.com/kubefirst/kubefirst-api/internal/gitopsCatalog"
	"github.com/kubefirst/kubefirst-api/internal/types"
	log "github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetGitopsCatalogApps
func (mdbcl *MongoDBClient) GetGitopsCatalogApps() (types.GitopsCatalogApps, error) {
	// Find
	var result types.GitopsCatalogApps
	err := mdbcl.GitopsCatalogCollection.FindOne(mdbcl.Context, bson.D{}).Decode(&result)
	if err != nil {
		return types.GitopsCatalogApps{}, fmt.Errorf("error getting gitops catalog apps: %s", err)
	}

	return result, nil
}

// UpdateGitopsCatalogApps
func (mdbcl *MongoDBClient) UpdateGitopsCatalogApps() error {
	mpapps, err := gitopsCatalog.ReadActiveApplications()
	if err != nil {
		log.Error().Msgf("error reading gitops catalog apps at startup: %s", err)
	}

	filter := bson.D{{Key: "name", Value: "gitops_catalog_application_list"}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "apps", Value: mpapps.Apps}}}}
	opts := options.Update().SetUpsert(true)

	_, err = mdbcl.GitopsCatalogCollection.UpdateOne(mdbcl.Context, filter, update, opts)
	if err != nil {
		return fmt.Errorf("error updating gitops catalog app list in database: %s", err)
	}
	log.Info().Msg("updated gitops catalog application directory")

	return nil
}
