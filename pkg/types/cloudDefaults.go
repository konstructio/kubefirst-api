package types

type CloudDefault struct {
	InstanceSize string `json:"instance_size"`
	NodeCount    int    `json:"node_count"`
}

type CloudProviderDefaults struct {
	Aws          CloudDefault `json:"aws"`
	Civo         CloudDefault `json:"civo"`
	DigitalOcean CloudDefault `json:"do"`
	Google       CloudDefault `json:"google"`
	Vultr        CloudDefault `json:"vultr"`
	K3d          CloudDefault `json:"k3d"`
}
