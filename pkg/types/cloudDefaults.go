package types

type CloudDefault struct {
	InstanceSize string `json:"instance_size"`
	NodeCount    string `json:"node_count"`
}

type CloudProviderDefaults struct {
	Akamai       CloudDefault `json:"akamai"`
	Aws          CloudDefault `json:"aws"`
	Azure        CloudDefault `json:"azure"`
	Civo         CloudDefault `json:"civo"`
	DigitalOcean CloudDefault `json:"do"`
	Google       CloudDefault `json:"google"`
	Vultr        CloudDefault `json:"vultr"`
	K3d          CloudDefault `json:"k3d"`
}
