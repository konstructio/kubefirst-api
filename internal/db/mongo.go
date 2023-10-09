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

	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/runtime/pkg/k8s"
	log "github.com/sirupsen/logrus"
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
	var connString string
	var clientOptions *options.ClientOptions

	ctx := context.Background()

	switch os.Getenv("MONGODB_HOST_TYPE") {
	case "atlas":
		serverAPI := options.ServerAPI(options.ServerAPIVersion1)
		connString = fmt.Sprintf("mongodb+srv://%s:%s@%s",
			os.Getenv("MONGODB_USERNAME"),
			os.Getenv("MONGODB_PASSWORD"),
			os.Getenv("MONGODB_HOST"),
		)
		clientOptions = options.Client().ApplyURI(connString).SetServerAPIOptions(serverAPI)
	case "local":
		connString = fmt.Sprintf("mongodb://%s:%s@%s/?authSource=admin",
			os.Getenv("MONGODB_USERNAME"),
			os.Getenv("MONGODB_PASSWORD"),
			os.Getenv("MONGODB_HOST"),
		)
		clientOptions = options.Client().ApplyURI(connString)
	}

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("could not create mongodb client: %s", err)
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
		log.Fatalf("error connecting to mongodb: %s", err)
	}
	if !silent {
		log.Infof("connected to mongodb host %s", os.Getenv("MONGODB_HOST"))
	}

	return nil
}

// ImportClusterIfEmpty
func (mdbcl *MongoDBClient) ImportClusterIfEmpty(silent bool, cloudProvider string) (pkgtypes.Cluster, error) {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "",
	})
	log.SetReportCaller(false)

	// find the secret in mgmt cluster's kubefirst namespace and read import payload and clustername
	var kcfg *k8s.KubernetesClient

	// homeDir, err := os.UserHomeDir()
	// if err != nil {
	// 	log.Fatalf("error getting home path: %s", err)
	// }

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("error getting home path: %s", err)
	}
	clusterDir := fmt.Sprintf("%s/.k1/%s", homeDir, "")

	inCluster := false
	if os.Getenv("IN_CLUSTER") == "true" {
		inCluster = true
	}

	kcfg = k8s.CreateKubeConfig(inCluster, fmt.Sprintf("%s/kubeconfig", clusterDir))

	log.Infof("reading secret mongo-state to determine if import is needed")
	secData, err := k8s.ReadSecretV2(kcfg.Clientset, "kubefirst", "mongodb-state")
	if err != nil {
		log.Infof("error reading secret mongodb-state. %s", err)
		return pkgtypes.Cluster{}, err
	}
	clusterName := secData["cluster-name"]
	importPayload := secData["cluster-0"]
	log.Infof("import cluster secret discovered for cluster %s", clusterName)

	// if you find a record bail
	// otherwise read the payload, import to db, bail

	filter := bson.D{{Key: "cluster_name", Value: clusterName}}
	// var result1 pkgtypes.Cluster
	var clusterFromSecret pkgtypes.Cluster
	//err = mdbcl.ClustersCollection.FindOne(mdbcl.Context, filter).Decode(&result1)
	err = mdbcl.ClustersCollection.FindOne(mdbcl.Context, filter).Decode(&clusterFromSecret)
	if err != nil {
		// This error means your query did not match any documents.
		log.Infof("did not find preexisting record for cluster %s. importing record.", clusterName)
		// clusterFromSecret := pkgtypes.Cluster{}
		unmarshalErr := bson.UnmarshalExtJSON([]byte(importPayload), true, &clusterFromSecret)
		if unmarshalErr != nil {
			log.Info("error encountered unmarshaling secret data")
			log.Error(unmarshalErr)
		}
		if err == mongo.ErrNoDocuments {
			// Create if entry does not exist
			_, err := mdbcl.ClustersCollection.InsertOne(mdbcl.Context, clusterFromSecret)
			if err != nil {
				return pkgtypes.Cluster{}, fmt.Errorf("error inserting cluster %s: %s", clusterName, err)
			}
			log.Info("inserted cluster record to db. adding default services.")

			return clusterFromSecret, nil
		} else {
			return pkgtypes.Cluster{}, fmt.Errorf("error inserting record: %s", err)
		}
	} else {
		log.Infof("cluster record for %s already exists - skipping", clusterName)
	}

	return pkgtypes.Cluster{}, nil
}

type EstablishConnectArgs struct {
	Tries  int
	Silent bool
}

func (mdbcl *MongoDBClient) EstablishMongoConnection(args EstablishConnectArgs) error {
	var pingError error

	for tries := 0; tries < args.Tries; tries += 1 {
		err := mdbcl.Client.Database("admin").RunCommand(mdbcl.Context, bson.D{{Key: "ping", Value: 1}}).Err()

		if err != nil {
			pingError = err
			fmt.Println("awaiting mongo db connectivity...")
			continue
		}

		if !args.Silent {
			log.Infof("connected to mongodb host %s", os.Getenv("MONGODB_HOST"))
		}

		return nil
	}

	return fmt.Errorf("unable to establish connection to mongo db: %s", pingError)
}
