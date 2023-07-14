/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/middleware"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// createApiUserCmd represents the generateApiUser command
var createApiUserCmd = &cobra.Command{
	Use:   "create-api-user",
	Short: "Create an API user",
	Long:  `Create an API user`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			log.Fatal("this command requires only a single argument - the name for the api user to be created")
		}

		apiUserName := args[0]

		// Check for required environment variables
		for _, v := range []string{"MONGODB_HOST", "MONGODB_USERNAME", "MONGODB_PASSWORD"} {
			if os.Getenv(v) == "" {
				log.Fatalf("the %s environment variable must be set", v)
			}
		}

		apiKey := middleware.GenerateAPIKey(16)
		err := db.Client.InsertUser(middleware.AuthorizedUser{
			Name:   apiUserName,
			APIKey: apiKey,
		})
		if err != nil {
			log.Fatalf("error creating user: %s", err)
		}

		log.Infof("created user %s with api key: %s", apiUserName, apiKey)
	},
}

func init() {
	rootCmd.AddCommand(createApiUserCmd)
}
