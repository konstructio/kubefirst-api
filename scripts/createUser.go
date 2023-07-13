package main

import (
	"crypto/rand"
	"encoding/hex"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/kubefirst/kubefirst-api/internal/db"
	"github.com/kubefirst/kubefirst-api/internal/middleware"
)

func main() {
	// Check for required environment variables
	if os.Getenv("MONGODB_HOST_TYPE") == "" {
		log.Fatalf("the MONGODB_HOST_TYPE environment variable must be set to either: atlas, local")
	}
	for _, v := range []string{"MONGODB_HOST", "MONGODB_USERNAME", "MONGODB_PASSWORD"} {
		if os.Getenv(v) == "" {
			log.Fatalf("the %s environment variable must be set", v)
		}
	}

	// Change user name here - API key will be automatically generated
	err := db.Client.InsertUser(middleware.AuthorizedUser{
		Name:   "myuser",
		APIKey: generateAPIKey(16),
	})
	if err != nil {
		log.Fatalf("error creating user: %s", err)
	}

	log.Infof("created user")
}

func generateAPIKey(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
