package linode

import (
	log "github.com/sirupsen/logrus"
)

// GetDNSInfo determines whether or not a domain exists within digitalocean
func (c *LinodeConfiguration) GetDNSInfo(domainName string) (string, error) {
	log.Info("GetDNSInfo (working...)")

	return "", nil
}
