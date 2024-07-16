package pkg

import (
	"math/rand"
	"time"

	internal "github.com/kubefirst/kubefirst-api/internal"
	helpers "github.com/kubefirst/kubefirst-api/internal/helpers"
)

func randSeq(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxy")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func Random(seq int) string {
	rand.Seed(time.Now().UnixNano())
	return randSeq(seq)
}

// internal exports
var IsAppAvailable = internal.IsAppAvailable
var GenerateClusterID = internal.GenerateClusterID
var GetAvailableDiskSize = internal.GetAvailableDiskSize
var OpenBrowser = internal.OpenBrowser
var ResetK1Dir = internal.ResetK1Dir
var SetupViper = internal.SetupViper
var OpenLogFile = internal.OpenLogFile

// helper exports
var DisplayLogHints = helpers.DisplayLogHints
var TestEndpointTLS = helpers.TestEndpointTLS
var SetClusterStatusFlags = helpers.SetClusterStatusFlags
var GetClusterStatusFlags = helpers.GetClusterStatusFlags
