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
)

// ValidateAPIKey determines whether or not a request is authenticated with a valid API key
func ValidateAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		APIKey := strings.TrimPrefix(c.Request.Header.Get("Authorization"), "Bearer ")

		if APIKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"status": 401, "message": "Authentication failed - no API key provided in request"})
			c.Abort()

			return
		}

		if APIKey != os.Getenv("K1_ACCESS_TOKEN") {
			c.JSON(http.StatusUnauthorized, gin.H{"status": 401, "message": "Authentication failed - not a valid API key"})
			c.Abort()

			return
		}
	}
}
