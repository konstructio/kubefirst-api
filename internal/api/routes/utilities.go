/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/kubefirst/kubefirst-api/internal/types"
)

var upgrader = websocket.Upgrader{}

// getHealth godoc
// @Summary Return health status if the application is running.
// @Description Return health status if the application is running.
// @Tags health
// @Produce json
// @Success 200 {object} types.JSONHealthResponse
// @Router /health [get]
func GetHealth(c *gin.Context) {
	c.JSON(http.StatusOK, types.JSONHealthResponse{
		Status: "healthy",
	})
}

// stream godoc
func Stream(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}
		log.Printf("Received message of type %d: %s", messageType, string(message))
	}
}
