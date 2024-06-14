package properties

import "fmt"

type CreateVM struct {
	AvailabilityZone  string             `json:"availability_zone"`
	BootFromVolume    *bool              `json:"boot_from_volume,omitempty"`
	EphemeralDisk     string             `json:"ephemeral_disk"`
	InstanceType      string             `json:"instance_type"`
	KeyName           string             `json:"key_name"`
	LoadbalancerPools []LoadbalancerPool `json:"loadbalancer_pools"`
	RootDisk          Disk               `json:"root_disk,omitempty"`
	SchedulerHints    string             `json:"scheduler_hints"`
	SecurityGroups    []string           `json:"security_groups"`
}

type Disk struct {
	Size int `json:"size"`
}

type LoadbalancerPool struct {
	Name           string `json:"name"`
	ProtocolPort   int    `json:"port"`
	MonitoringPort *int   `json:"monitoring_port,omitempty"`
}

func (c CreateVM) Validate() error {

	for _, pool := range c.LoadbalancerPools {
		if pool.Name == "" {
			return fmt.Errorf("load balancer pool defined without name")
		}
		if pool.ProtocolPort == 0 {
			return fmt.Errorf("load balancer pool '%s' has no port definition", pool.Name)
		}
	}
	return nil
}

type CreateVMNetwork struct {
	NetID          string   `json:"net_id"`
	SecurityGroups []string `json:"security_groups"`
}
