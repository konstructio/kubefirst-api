package segment

import (
	"fmt"

	"github.com/segmentio/analytics-go"
)

// SendCountMetric
func (c *SegmentClient) SendCountMetric(
	metricName string,
	errorMessage string,
) string {
	if metricName == MetricInitStarted {
		err := c.Client.Enqueue(analytics.Identify{
			UserId: c.DomainName,
			Type:   "identify",
		})
		if err != nil {
			return fmt.Sprintf("error sending identify to segment: %s", err.Error())
		}
	}
	err := c.Client.Enqueue(analytics.Track{
		UserId: c.DomainName,
		Event:  metricName,
		Properties: analytics.NewProperties().
			Set("cli_version", c.CliVersion).
			Set("cloud_provider", c.CloudProvider).
			Set("cluster_id", c.ClusterID).
			Set("cluster_type", c.ClusterType).
			Set("domain", c.DomainName).
			Set("git_provider", c.GitProvider).
			Set("client", c.KubefirstClient).
			Set("kubefirst_team", c.KubefirstTeam).
			Set("kubefirst_team_info", c.KubefirstTeamInfo).
			Set("machine_id", c.MachineID).
			Set("error", errorMessage).
			Set("install_method", c.InstallMethod),
	})
	if err != nil {
		return fmt.Sprintf("error sending track to segment: %s", err.Error())
	}

	return ""
}
