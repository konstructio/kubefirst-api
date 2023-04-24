/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/types"
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
// @Router /stream [get]
// GetLogs
func GetLogs(c *gin.Context) {
	setHeaders(c)

	// Stream logs
	logs := make(chan types.LogMessage)
	done := make(chan struct{})
	errCh := make(chan error)
	go func() {
		err := StreamLogs(logs, errCh, done)
		if err != nil {
			errCh <- err
		}
	}()

	// Stream logs to client using SSE
	streamLogsToClient(c, logs, errCh, done)
}

// StreamLogs redirects stdout logs to the stream via SSE
func StreamLogs(ch chan types.LogMessage, errCh chan error, done chan struct{}) error {
	r, w, _ := os.Pipe()
	os.Stdout = w

	go func(reader io.Reader) {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			ch <- types.LogMessage{
				Message: scanner.Text(),
			}
		}
		if err := scanner.Err(); err != nil {
			errCh <- fmt.Errorf("error during log stream")

		}
	}(r)

	return nil
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
