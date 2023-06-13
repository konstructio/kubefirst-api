/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package db

import (
	"fmt"

	"github.com/kubefirst/kubefirst-api/internal/marketplace"
	"github.com/kubefirst/kubefirst-api/internal/types"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

// GetMarketplaceApps
func (mdbcl *MongoDBClient) GetMarketplaceApps() (types.MarketplaceApps, error) {
	// Find
	var result types.MarketplaceApps
	err := mdbcl.MarketplaceCollection.FindOne(mdbcl.Context, bson.D{}).Decode(&result)
	if err != nil {
		return types.MarketplaceApps{}, fmt.Errorf("error getting marketplace apps: %s", err)
	}

	return result, nil
}

// InsertMarketplaceApps
func (mdbcl *MongoDBClient) InsertMarketplaceApps() error {
	mpapps, err := marketplace.ReadActiveApplications()
	if err != nil {
		log.Errorf("error reading marketplace apps at startup: %s", err)
	}

	// todo: this should compare and replace
	insert, err := mdbcl.MarketplaceCollection.InsertOne(mdbcl.Context, mpapps)
	if err != nil {
		return fmt.Errorf("error updating marketplace app list in database: %s", err)
	}
	log.Info("updated marketplace application directory: %s", insert)

	return nil
}
