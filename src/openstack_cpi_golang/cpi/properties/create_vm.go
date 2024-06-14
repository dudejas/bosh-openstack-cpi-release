package properties

type CreateVM struct {
	AvailabilityZone  string   `json:"availability_zone"`
	BootFromVolume    *bool    `json:"boot_from_volume,omitempty"`
	EphemeralDisk     string   `json:"ephemeral_disk"`
	InstanceType      string   `json:"instance_type"`
	KeyName           string   `json:"key_name"`
	LoadbalancerPools string   `json:"loadbalancer_pools"`
	RootDisk          Disk     `json:"root_disk,omitempty"`
	SchedulerHints    string   `json:"scheduler_hints"`
	SecurityGroups    []string `json:"security_groups"`
}

type Disk struct {
	Size int `json:"size"`
}

type CreateVMNetwork struct {
	NetID          string   `json:"net_id"`
	SecurityGroups []string `json:"security_groups"`
}
