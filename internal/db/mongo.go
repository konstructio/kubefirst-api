/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package db

import (
	"context"
	"fmt"
	"os"

	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/env"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/runtime/pkg/k8s"
	log "github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBClient struct {
	Client                  *mongo.Client
	Context                 context.Context
	ClustersCollection      *mongo.Collection
	GitopsCatalogCollection *mongo.Collection
	ServicesCollection      *mongo.Collection
	EnvironmentsCollection  *mongo.Collection
}

var Client = Connect()

// 1 Client, Mongo not ready

// Connect
func Connect() *MongoDBClient {
	env, getEnvError := env.GetEnv(constants.SilenceGetEnv)

	if getEnvError != nil {
		log.Fatal().Msg(getEnvError.Error())
	}

	var connString string
	var clientOptions *options.ClientOptions

	ctx := context.Background()

	switch env.MongoDBHostType {
	case "atlas":
		serverAPI := options.ServerAPI(options.ServerAPIVersion1)
		connString = fmt.Sprintf("mongodb+srv://%s:%s@%s",
			env.MongoDBUsername,
			env.MongoDBPassword,
			env.MongoDBHost,
		)
		clientOptions = options.Client().ApplyURI(connString).SetServerAPIOptions(serverAPI)
	case "local":
		connString = fmt.Sprintf("mongodb://%s:%s@%s/?authSource=admin",
			env.MongoDBUsername,
			env.MongoDBPassword,
			env.MongoDBHost,
		)
		clientOptions = options.Client().ApplyURI(connString)
	}

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal().Msgf("could not create mongodb client: %s", err)
	}

	cl := MongoDBClient{
		Client:                  client,
		Context:                 ctx,
		ClustersCollection:      client.Database("api").Collection("clusters"),
		GitopsCatalogCollection: client.Database("api").Collection("gitops-catalog"),
		ServicesCollection:      client.Database("api").Collection("services"),
		EnvironmentsCollection:  client.Database("api").Collection("environments"),
	}

	return &cl
}

// TestDatabaseConnection
func (mdbcl *MongoDBClient) TestDatabaseConnection(silent bool) error {
	err := mdbcl.Client.Database("admin").RunCommand(mdbcl.Context, bson.D{{Key: "ping", Value: 1}}).Err()
	if err != nil {
		log.Fatal().Msgf("error connecting to mongodb: %s", err)
	}
	if !silent {
		env, _ := env.GetEnv(constants.SilenceGetEnv)

		log.Info().Msgf("connected to mongodb host %s", env.MongoDBHost)
	}

	return nil
}

// ImportClusterIfEmpty
func (mdbcl *MongoDBClient) ImportClusterIfEmpty(silent bool) (pkgtypes.Cluster, error) {
	env, _ := env.GetEnv(constants.SilenceGetEnv)

	// find the secret in mgmt cluster's kubefirst namespace and read import payload and clustername
	var kcfg *k8s.KubernetesClient

	var isClusterZero bool = true
	if env.IsClusterZero == "false" {
		isClusterZero = false
	}

	if isClusterZero {
		log.Info().Msg("IS_CLUSTER_ZERO is set to true, skipping import cluster logic.")
		return pkgtypes.Cluster{}, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal().Msgf("error getting home path: %s", err)
	}
	clusterDir := fmt.Sprintf("%s/.k1/%s", homeDir, "")

	var inCluster bool = false
	if env.InCluster == "true" {
		inCluster = true
	}

	kcfg = k8s.CreateKubeConfig(inCluster, fmt.Sprintf("%s/kubeconfig", clusterDir))

	log.Info().Msg("reading secret mongo-state to determine if import is needed")
	secData, err := k8s.ReadSecretV2(kcfg.Clientset, "kubefirst", "mongodb-state")
	if err != nil {
		log.Info().Msgf("error reading secret mongodb-state. %s", err)
		return pkgtypes.Cluster{}, err
	}
	clusterName := secData["cluster-name"]
	importPayload := secData["cluster-0"]
	log.Info().Msgf("import cluster secret discovered for cluster %s", clusterName)

	// if you find a record bail
	// otherwise read the payload, import to db, bail

	filter := bson.D{{Key: "cluster_name", Value: clusterName}}
	// var result1 pkgtypes.Cluster
	var clusterFromSecret pkgtypes.Cluster
	//err = mdbcl.ClustersCollection.FindOne(mdbcl.Context, filter).Decode(&result1)
	err = mdbcl.ClustersCollection.FindOne(mdbcl.Context, filter).Decode(&clusterFromSecret)
	if err != nil {
		// This error means your query did not match any documents.
		log.Info().Stack().Msgf("did not find preexisting record for cluster %s. importing record.", clusterName)
		// clusterFromSecret := pkgtypes.Cluster{}
		unmarshalErr := bson.UnmarshalExtJSON([]byte(importPayload), true, &clusterFromSecret)
		if unmarshalErr != nil {
			log.Info().Msg("error encountered unmarshaling secret data")
			log.Error().Msg(unmarshalErr.Error())
		}
		if err == mongo.ErrNoDocuments {
			// Create if entry does not exist
			_, err := mdbcl.ClustersCollection.InsertOne(mdbcl.Context, clusterFromSecret)
			if err != nil {
				return pkgtypes.Cluster{}, fmt.Errorf("error inserting cluster %v: %s", clusterFromSecret, err)
			}
			// log clusterFromSecret
			log.Info().Msgf("inserted cluster record to db. adding default services. %s", clusterFromSecret.ClusterName)

			return clusterFromSecret, nil
		} else {
			return pkgtypes.Cluster{}, fmt.Errorf("error inserting record: %s", err)
		}
	} else {
		log.Info().Msgf("cluster record for %s already exists - skipping", clusterName)
	}

	return pkgtypes.Cluster{}, nil
}

type EstablishConnectArgs struct {
	Tries  int
	Silent bool
}

func (mdbcl *MongoDBClient) EstablishMongoConnection(args EstablishConnectArgs) error {
	env, _ := env.GetEnv(constants.SilenceGetEnv)

	var pingError error

	for tries := 0; tries < args.Tries; tries += 1 {
		err := mdbcl.Client.Database("admin").RunCommand(mdbcl.Context, bson.D{{Key: "ping", Value: 1}}).Err()

		if err != nil {
			pingError = err
			fmt.Println("awaiting mongo db connectivity...")
			continue
		}

		if !args.Silent {
			log.Info().Msgf("connected to mongodb host %s", env.MongoDBHost)
		}

		return nil
	}

	return fmt.Errorf("unable to establish connection to mongo db: %s", pingError)
}
