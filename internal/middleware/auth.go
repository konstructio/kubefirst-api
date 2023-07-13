/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// ValidateAPIKey determines whether or not a request is authenticated with a valid API key
func ValidateAPIKey(users *mongo.Collection) gin.HandlerFunc {
	return func(c *gin.Context) {
		var user AuthorizedUser

		APIKey := strings.TrimPrefix(c.Request.Header.Get("Authorization"), "Bearer ")

		if APIKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"status": 401, "message": "Authentication failed - no API key provided in request"})
			c.Abort()

			return
		}

		filter := bson.D{{Key: "api_key", Value: APIKey}}
		if err := users.FindOne(context.Background(), filter).Decode(&user); err != nil {
			fmt.Println(user)
			c.JSON(http.StatusUnauthorized, gin.H{"status": 401, "message": "Authentication failed - not a valid API key"})
			c.Abort()

			return
		}
	}
}
