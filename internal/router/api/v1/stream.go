/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/nxadm/tail"
)

// setHeaders sets headers for the SSE response
func setHeaders(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
}

// GetLogs godoc
// @Summary Stream API server logs
// @Description Stream API server logs
// @Tags logs
// @Router /stream/file_name [get]
// @Param Authorization header string true "API key" default(Bearer <API key>)
// GetLogs
func GetLogs(c *gin.Context) {
	setHeaders(c)

	fileName, param := c.Params.Get("file_name")

	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":file_name not provided",
		})
		return
	}

	// Stream logs
	if err := StreamLogs(c, fileName); err != nil {
		c.SSEvent("error", err.Error())
		c.Writer.Flush()
	}
}

// StreamLogs redirects stdout logs to the stream via SSE
func StreamLogs(c *gin.Context, fileName string) error {
	homePath, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error getting current user's home directory: %w", err)
	}

	logfile := filepath.Join(homePath, ".k1", "logs", fileName)

	t, err := tail.TailFile(logfile, tail.Config{Follow: true, ReOpen: true})
	if err != nil {
		return fmt.Errorf("error opening log file %q: %w", logfile, err)
	}
	defer t.Cleanup()

	for {
		select {
		case <-c.Request.Context().Done():
			// if the request itself has been closed, then we just continue
			return nil

		case <-c.Writer.CloseNotify():
			// if the place where we're writing the logs has been closed, then we just continue
			return nil

		case line, ok := <-t.Lines:
			if !ok {
				// channel has been closed, nothing more to do
				return nil
			}

			c.SSEvent("", line.Text)
			c.Writer.Flush()

			// Sleep for a short time to avoid overwhelming the client
			time.Sleep(100 * time.Millisecond)
		}
	}
}
