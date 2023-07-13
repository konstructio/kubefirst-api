/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package db

import (
	"fmt"

	"github.com/kubefirst/kubefirst-api/internal/middleware"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// InsertUser
func (mdbcl *MongoDBClient) InsertUser(user middleware.AuthorizedUser) error {
	filter := bson.D{primitive.E{Key: "Name", Value: user.Name}}

	var result middleware.AuthorizedUser

	err := mdbcl.UsersCollection.FindOne(mdbcl.Context, filter).Decode(&result)
	if err != nil {
		// This error means your query did not match any documents.
		if err == mongo.ErrNoDocuments {
			// Create if entry does not exist
			user = middleware.AuthorizedUser{
				Name:   user.Name,
				APIKey: user.APIKey,
			}

			_, err := mdbcl.UsersCollection.InsertOne(mdbcl.Context, user)
			if err != nil {
				return fmt.Errorf("error inserting user %s: %s", user.Name, err)
			}
		}
	} else {
		log.Infof("user %s already exists - skipping", user.Name)
	}

	return nil
}
