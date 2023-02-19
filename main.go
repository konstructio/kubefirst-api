/*
Copyright © 2023 Kubefirst <kubefirst.io>
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package main

import (
	"fmt"

	"github.com/kubefirst/kubefirst-api/docs"
	"github.com/kubefirst/kubefirst-api/internal/api"

	log "github.com/sirupsen/logrus"
)

// @title Kubefirst API
// @version 1.0
// @description Kubefirst API
// @contact.name Kubefirst
// @contact.email kubefirst@kubefirst.com
// @host localhost:port
// @BasePath /api/v1

const (
	port int = 8080
)

func main() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "",
	})
	log.SetReportCaller(false)

	// Programmatically set swagger info
	docs.SwaggerInfo.Title = "Kubefirst API"
	docs.SwaggerInfo.Description = "Kubefirst API"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%v", port)
	docs.SwaggerInfo.BasePath = "/api/v1"
	docs.SwaggerInfo.Schemes = []string{"http"}

	// API
	r := api.SetupRouter()
	err := r.Run(fmt.Sprintf(":%v", port))
	if err != nil {
		log.Fatalf("Error starting API: %s", err)
	}
}
