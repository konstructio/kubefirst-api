package segment

import (
	"fmt"

	"github.com/denisbrodbeck/machineid"
	"github.com/kubefirst/runtime/pkg"
	"github.com/segmentio/analytics-go"
)

// SendCountMetric
func (c *SegmentClient) SendCountMetric(
	metricName string,
	errorMessage string,
) string {
	strippedDomainName, err := pkg.RemoveSubdomainV2(c.DomainName)
	if err != nil {
		return "error stripping domain name from value"
	}
	machineID, _ := machineid.ID()
	if metricName == MetricInitStarted {
		err := c.Client.Enqueue(analytics.Identify{
			UserId: strippedDomainName,
			Type:   "identify",
		})
		if err != nil {
			return fmt.Sprintf("error sending identify to segment: %s", err.Error())
		}
	}
	err = c.Client.Enqueue(analytics.Track{
		UserId: strippedDomainName,
		Event:  metricName,
		Properties: analytics.NewProperties().
			Set("cli_version", c.CliVersion).
			Set("cloud_provider", c.CloudProvider).
			Set("cluster_id", c.ClusterID).
			Set("cluster_type", c.ClusterType).
			Set("domain", strippedDomainName).
			Set("git_provider", c.GitProvider).
			Set("client", c.KubefirstClient).
			Set("kubefirst_team", c.KubefirstTeam).
			Set("kubefirst_team_info", c.KubefirstTeamInfo).
			Set("machine_id", machineID).
			Set("error", errorMessage).
			Set("install_method", c.InstallMethod),
	})
	if err != nil {
		return fmt.Sprintf("error sending track to segment: %s", err.Error())
	}

	return ""
}
