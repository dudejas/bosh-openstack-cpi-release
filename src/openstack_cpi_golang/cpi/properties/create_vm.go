package properties

type CreateVM struct {
	AvailabilityZone string `json:"availability_zone"`
	EphemeralDisk    string `json:"ephemeral_disk"`
	InstanceType     string `json:"instance_type"`
}

type CreateVMNetwork struct {
	NetID          string   `json:"net_id"`
	SecurityGroups []string `json:"security_groups"`
}
