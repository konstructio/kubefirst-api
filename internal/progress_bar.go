/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package internal

import (
	"fmt"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
)

type ActionTracker struct {
	Tracker *progress.Tracker
}

const (
	DownloadDependencies               = "Download dependencies"
	GetAccountInfo                     = "Get Account Info"
	GetDNSInfo                         = "Get DNS Info"
	TestHostedZoneLiveness             = "Test Domain Liveness"
	CloneAndDetokenizeGitopsTemplate   = "Clone and Detokenize (Gitops)"
	CloneAndDetokenizeMetaphorTemplate = "Clone and Detokenize (Metaphor)"
	CreateSSHKey                       = "Create SSH keys"
	CreateBuckets                      = "Create Buckets"
	SendTelemetry                      = "Send Telemetry"
)

var (
	pw       progress.Writer
	Trackers map[string]*ActionTracker
)

// GetTrackers keeps one single instance of Trackers alive using singleton pattern.
func GetTrackers() map[string]*ActionTracker {
	if Trackers != nil {
		return Trackers
	}
	Trackers = make(map[string]*ActionTracker)
	return Trackers
}

// CreateTracker receives Tracker data and return a new Tracker.
func CreateTracker(title string, total int64) *progress.Tracker {
	tracker := &progress.Tracker{
		Message: title,
		Total:   total,
		Units:   progress.UnitsDefault,
	}

	pw.AppendTracker(tracker)

	return tracker
}

// SetupProgress prepare the progress bar setting its initial configuration
func SetupProgress(numTrackers int) {
	fmt.Printf("Init actions: %d expected tasks ...\n\n", numTrackers)
	// instantiate a Progress Writer and set up the options
	pw = progress.NewWriter()
	pw.SetAutoStop(false)
	pw.SetTrackerLength(40)
	pw.SetMessageLength(39)
	pw.SetNumTrackersExpected(numTrackers)
	pw.SetSortBy(progress.SortByPercentDsc)
	pw.SetStyle(progress.StyleDefault)
	pw.SetTrackerPosition(progress.PositionRight)
	pw.SetUpdateFrequency(time.Millisecond * 100)
	pw.Style().Colors = progress.StyleColorsExample
	pw.Style().Options.PercentFormat = "%4.1f%%"
	pw.Style().Visibility.ETA = true
	pw.Style().Visibility.ETAOverall = true
	pw.Style().Visibility.Percentage = true
	pw.Style().Visibility.Time = true
	pw.Style().Visibility.TrackerOverall = true
	pw.Style().Visibility.Value = true
	go pw.Render()
}
