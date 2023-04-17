/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package db

import (
	"context"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBClient struct {
	Client     *mongo.Client
	Collection *mongo.Collection
	Context    context.Context
}

// InitDatabase
func (mdbcl *MongoDBClient) InitDatabase() error {
	mdbcl.Context = context.Background()

	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017/")
	client, err := mongo.Connect(mdbcl.Context, clientOptions)
	if err != nil {
		return err
	}

	err = client.Ping(mdbcl.Context, nil)
	if err != nil {
		return err
	}

	mdbcl.Client = client
	mdbcl.Collection = client.Database("api").Collection("clusters")

	return nil
}

// InsertCluster
func (mdbcl *MongoDBClient) InsertCluster(cl Cluster) error {
	filter := bson.D{{"cluster_name", cl.ClusterName}}
	var result Cluster
	err := mdbcl.Collection.FindOne(mdbcl.Context, filter).Decode(&result)
	if err != nil {
		// This error means your query did not match any documents.
		if err == mongo.ErrNoDocuments {
			// Create if entry does not exist
			insert, err := mdbcl.Collection.InsertOne(mdbcl.Context, cl)
			if err != nil {
				return err
			}
			log.Info(insert)
		}
	} else {
		log.Infof("cluster record for %s already exists - skipping", cl.ClusterName)
	}

	return nil
}

// GetCluster
func (mdbcl *MongoDBClient) GetCluster(clusterName string) (Cluster, error) {
	// Find
	filter := bson.D{{"cluster_name", clusterName}}
	var result Cluster
	err := mdbcl.Collection.FindOne(mdbcl.Context, filter).Decode(&result)
	if err != nil {
		return Cluster{}, err
	}

	return result, nil
}

// UpdateCluster
func (mdbcl *MongoDBClient) UpdateCluster(clusterName string, field string, value interface{}) error {
	// Find
	filter := bson.D{{"cluster_name", clusterName}}
	var result Cluster
	err := mdbcl.Collection.FindOne(mdbcl.Context, filter).Decode(&result)
	if err != nil {
		return err
	}

	// Update
	filter = bson.D{{"_id", result.ID}}
	update := bson.D{{"$set", bson.D{{field, value}}}}
	resp, err := mdbcl.Collection.UpdateOne(mdbcl.Context, filter, update)
	if err != nil {
		return err
	}

	log.Infof("Documents updated: %v\n", resp.ModifiedCount)

	return nil
}

// Cluster describes the configuration storage for a Kubefirst cluster object
type Cluster struct {
	ID primitive.ObjectID `bson:"_id"`

	ClusterName   string `bson:"cluster_name"`
	CloudProvider string `bson:"cloud_provider"`
	CloudRegion   string `bson:"cloud_region"`
	DomainName    string `bson:"domain_name"`
	ClusterID     string `bson:"cluster_id"`
	ClusterType   string `bson:"cluster_type"`

	GitProvider        string `bson:"git_provider"`
	GitHost            string `bson:"git_host"`
	GitOwner           string `bson:"git_owner"`
	GitUser            string `bson:"git_user"`
	GitToken           string `bson:"git_token"`
	GitlabOwnerGroupID int    `bson:"gitlab_owner_group_id"`

	AtlantisWebhookSecret string `bson:"atlantis_webhook_secret"`
	KubefirstTeam         string `bson:"kubefirst_team"`

	PublicKey  string `bson:"public_key"`
	PrivateKey string `bson:"private_key"`
	PublicKeys string `bson:"public_keys"`

	ArgoCDUsername  string `bson:"argocd_username"`
	ArgoCDPassword  string `bson:"argocd_password"`
	ArgoCDAuthToken string `bson:"argocd_auth_token"`

	// Checks
	KbotSetupCheck                 bool `bson:"kbot_setup_check"`
	GitCredentialsCheck            bool `bson:"git_credentials_check"`
	GitopsReadyCheck               bool `bson:"gitops_ready_check"`
	GitTerraformApplyCheck         bool `bson:"git_terraform_apply_check"`
	GitopsPushedCheck              bool `bson:"gitops_pushed_check"`
	CloudTerraformApplyCheck       bool `bson:"cloud_terraform_apply_check"`
	CloudTerraformApplyFailedCheck bool `bson:"cloud_terraform_apply_failed_check"`
	ClusterSecretsCreatedCheck     bool `bson:"cluster_secrets_created_check"`
	ArgoCDInstallCheck             bool `bson:"argocd_install_check"`
	ArgoCDInitializeCheck          bool `bson:"argocd_initialize_check"`
	ArgoCDCreateRegistryCheck      bool `bson:"argocd_create_registry_check"`
	VaultInitializedCheck          bool `bson:"vault_initialized_check"`
	VaultTerraformApplyCheck       bool `bson:"vault_terraform_apply_check"`
	UsersTerraformApplyCheck       bool `bson:"users_terraform_apply_check"`
	PostDetokenizeCheck            bool `bson:"post_detokenize_check"`
}
