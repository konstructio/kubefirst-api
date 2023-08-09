/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// // GenerateAPIKey generates a random string to serve as an API key
// func GenerateAPIKey(length int) string {
// 	b := make([]byte, length)
// 	if _, err := rand.Read(b); err != nil {
// 		return ""
// 	}

// 	return hex.EncodeToString(b)
// }

// ValidateAPIKey determines whether or not a request is authenticated with a valid API key
func ValidateAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		// var user AuthorizedUser
		log.Info("ValidateAPIKey called")
		APIKey := strings.TrimPrefix(c.Request.Header.Get("Authorization"), "Bearer ")
		// log.Info("APIKey", APIKey)

		if APIKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"status": 401, "message": "Authentication failed - no API key provided in request"})
			c.Abort()

			return
		}

		// filter := bson.D{{Key: "api_key", Value: APIKey}}
		if APIKey == os.Getenv("K1_ACCESS_TOKEN") {
			c.JSON(http.StatusUnauthorized, gin.H{"status": 401, "message": "Authentication failed - not a valid API key"})
			c.Abort()

			return
		}
	}
}
