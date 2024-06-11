package properties

type CreateVM struct {
	AvailabilityZone  string   `json:"availability_zone"`
	EphemeralDisk     string   `json:"ephemeral_disk"`
	InstanceType      string   `json:"instance_type"`
	LoadbalancerPools string   `json:"loadbalancer_pools"`
	SchedulerHints    string   `json:"scheduler_hints"`
	SecurityGroups    []string `json:"security_groups"`
	KeyName           string   `json:"key_name"`
}

type CreateVMNetwork struct {
	NetID          string   `json:"net_id"`
	SecurityGroups []string `json:"security_groups"`
}

type NetworkConfig struct {
	DefaultNetwork Network
	ManualNetworks []Network
	VIPNetwork     *Network
	DynamicNetwork *Network
	SecurityGroups []string
}

type Network struct {
	Type       string
	CloudProps CreateVMNetwork
	IP         string
}
