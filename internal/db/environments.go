package db

import (
	"fmt"

	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// GetEnvironments
func (mdbcl *MongoDBClient) GetEnvironments() ([]pkgtypes.Environment, error) {
	// Find
	var result []pkgtypes.Environment
	cursor, err := mdbcl.EnvironmentsCollection.Find(mdbcl.Context, bson.D{})
	if err != nil {
		return []pkgtypes.Environment{}, fmt.Errorf("error getting environments")
	}

	for cursor.Next(mdbcl.Context) {
		//Create a value into which the single document can be decoded
		var environment pkgtypes.Environment
		err := cursor.Decode(&environment)
		if err != nil {
			return []pkgtypes.Environment{}, err
		}
		result = append(result, environment)

	}
	if err := cursor.Err(); err != nil {
		return []pkgtypes.Environment{}, err
	}

	cursor.Close(mdbcl.Context)

	return result, nil
}

// GetEnvironment
func (mdbcl *MongoDBClient) GetEnvironment(name string) (pkgtypes.Environment, error) {
	// Find
	filter := bson.D{{Key: "name", Value: name }}
	var result pkgtypes.Environment
	err := mdbcl.EnvironmentsCollection.FindOne(mdbcl.Context, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return pkgtypes.Environment{}, fmt.Errorf("environment not found")
		}
		return pkgtypes.Environment{}, fmt.Errorf("error getting environment %s: %s", name, err)
	}

	return result, nil
}

// InsertEnvironment
func (mdbcl *MongoDBClient) InsertEnvironment(env pkgtypes.Environment) (pkgtypes.Environment ,error) {
	filter := bson.D{{ Key: "name", Value: env.Name }}

	result := pkgtypes.Environment {
		ID: primitive.NewObjectID(),
		Name: env.Name,
		Color: env.Color,
		Description: env.Description,
		CreationTimestamp: env.CreationTimestamp,
	}

	err := mdbcl.EnvironmentsCollection.FindOne(mdbcl.Context, filter).Decode(&result)
	if err != nil {
		// This error means your query did not match any documents.
		if err == mongo.ErrNoDocuments {
			// Create if entry does not exist
			insert, err := mdbcl.EnvironmentsCollection.InsertOne(mdbcl.Context, result)
			if err != nil {
				return pkgtypes.Environment{}, fmt.Errorf("error inserting environment %v: %s", env.Name, err)
			}

			log.Info(insert)
		}
	} else {
		return pkgtypes.Environment{}, fmt.Errorf("environment %v already exists", env.Name)
	}
	return result, nil
}

func (mdbcl *MongoDBClient) DeleteEnvironment(envName string) error {
	filter := bson.D{{Key: "name", Value: envName }}

	findError := mdbcl.EnvironmentsCollection.FindOne(mdbcl.Context, filter).Err()

	if findError != nil {
		return fmt.Errorf("no environment by the name %v", envName)
	}

	_,err := mdbcl.EnvironmentsCollection.DeleteOne(mdbcl.Context, filter)
	if err != nil {
		return fmt.Errorf("error deleting environment %s: %s", envName, err)
	}

	log.Infof("%v environment deleted", envName)

	return nil
}