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
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/nxadm/tail"
	log "github.com/rs/zerolog/log"
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

	var (
		mu     sync.Mutex
		buffer string
	)

	fileName, param := c.Params.Get("file_name")

	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":file_name not provided",
		})
		return
	}

	// Stream logs
	logs := make(chan types.LogMessage)
	done := make(chan struct{})
	errCh := make(chan error)
	go func() {
		err := StreamLogs(fileName, logs, errCh, done, mu, buffer)
		if err != nil {
			errCh <- err
		}
	}()

	// Stream logs to client using SSE
	streamLogsToClient(c, logs, errCh, done)
}

// StreamLogs redirects stdout logs to the stream via SSE
func StreamLogs(fileName string, ch chan types.LogMessage, errCh chan error, done chan struct{}, mu sync.Mutex, buffer string) error {
	homePath, err := os.UserHomeDir()
	if err != nil {
		log.Info().Msg(err.Error())
	}
	k1Dir := fmt.Sprintf("%s/.k1", homePath)

	// * create log directory
	logsFolder := fmt.Sprintf("%s/logs", k1Dir)

	logfile := fmt.Sprintf("%s/%s", logsFolder, fileName)

	t, err := tail.TailFile(logfile, tail.Config{Follow: true, ReOpen: true})
	if err != nil {
		return fmt.Errorf("error opening log file %s", err.Error())
	}

	// Continuously stream log lines to the client
	for {
		select {
		case line, ok := <-t.Lines:
			if !ok {
				return err
			}
			mu.Lock()
			buffer = line.Text
			mu.Unlock()

			// Send the log line to the client as an event
			ch <- types.LogMessage{
				Message: line.Text,
			}

			// Sleep for a short time to avoid overwhelming the client
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// streamLogsToClient
func streamLogsToClient(c *gin.Context, logs chan types.LogMessage, errCh chan error, done chan struct{}) {
	for {
		select {
		// received new log line in go channel
		case log := <-logs:
			c.SSEvent(log.Type, log)
			c.Writer.Flush()
		case err := <-errCh:
			c.SSEvent("error", err.Error())
			return
			// channel should be closed
		case <-c.Writer.CloseNotify():
			close(done)
			return
		}
	}
}
