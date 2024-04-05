package config

import (
	"encoding/json"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type CpiConfig struct {
	Cloud struct {
		Properties struct {
			Openstack OpenstackConfig `json:"openstack"`
		} `json:"properties"`
	} `json:"cloud"`
}

type OpenstackConfig struct {
	AuthURL                      string   `json:"auth_url"`
	Username                     string   `json:"username"`
	APIKey                       string   `json:"api_key"`
	Region                       string   `json:"region"`
	EndpointType                 string   `json:"endpoint_type"`
	DefaultKeyName               string   `json:"default_key_name"`
	DefaultSecurityGroups        []string `json:"default_security_groups"`
	DefaultVolumeType            string   `json:"default_volume_type"`
	WaitResourcePollInterval     int      `json:"wait_resource_poll_interval"`
	BootFromVolume               bool     `json:"boot_from_volume"`
	ConfigDrive                  string   `json:"config_drive"`
	UseDhcp                      bool     `json:"use_dhcp"`
	IgnoreServerAvailabilityZone bool     `json:"ignore_server_availability_zone"`
	HumanReadableVMNames         bool     `json:"human_readable_vm_names"`
	UseNovaNetworking            bool     `json:"use_nova_networking"`
	ConnectionOptions            string   `json:"connection_options"`
	DomainName                   string   `json:"domain"`
	ProjectName                  string   `json:"project"`
	Tenant                       string   `json:"tenant"`
	StateTimeOut                 int      `json:"state_timeout"`
	StemcellPublicVisibility     bool     `json:"stemcell_public_visibility"`
	VM                           struct {
		Stemcell struct {
			APIVersion int `json:"api_version"`
		} `json:"stemcell"`
	} `json:"vm"`
}

func (cpiConfig CpiConfig) Validate() error {
	err := cpiConfig.Cloud.Properties.Openstack.Validate()
	if err != nil {
		return bosherr.WrapError(err, "Validating Config configuration")
	}

	return nil
}

func (openstackConfig OpenstackConfig) Validate() error {
	// do validation here

	return nil
}

func NewConfigFromPath(path string, fs boshsys.FileSystem) (CpiConfig, error) {
	var config CpiConfig

	bytes, err := fs.ReadFile(path)
	if err != nil {
		return config, bosherr.WrapErrorf(err, "Reading config '%s'", path)
	}

	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return config, bosherr.WrapError(err, "Unmarshalling config")
	}

	err = config.Validate()
	if err != nil {
		return config, bosherr.WrapError(err, "Validating config")
	}

	return config, nil
}
