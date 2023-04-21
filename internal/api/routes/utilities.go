/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/kubefirst/kubefirst-api/internal/types"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow cross-origin access from your React app
		return r.Header.Get("Origin") == "http://localhost:3000"
	},
}

// getHealth godoc
// @Summary Return health status if the application is running.
// @Description Return health status if the application is running.
// @Tags health
// @Produce json
// @Success 200 {object} types.JSONHealthResponse
// @Router /health [get]
func GetHealth(c *gin.Context) {
	c.JSON(http.StatusOK, types.JSONHealthResponse{
		Status: "healthz",
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

	file, err := os.Open("example-k3d.log")
	if err != nil {
		log.Printf("Failed to open log file: %v", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if err := conn.WriteMessage(websocket.TextMessage, []byte(line)); err != nil {
			log.Printf("WebSocket write error: %v", err)
			return
		}
		time.Sleep(time.Second * 4) // simulate a delay between logs
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Failed to read log file: %v", err)
		return
	}
}
