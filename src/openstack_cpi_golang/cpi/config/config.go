package config

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"strings"
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
	StemcellPubliclyVisible      bool     `json:"stemcell_public_visibility"`
	VM                           struct {
		Stemcell struct {
			APIVersion int `json:"api_version"`
		} `json:"stemcell"`
	} `json:"vm"`
}

func (cpiConfig CpiConfig) Validate() error {
	err := cpiConfig.Cloud.Properties.Openstack.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate the configuration: %w", err)
	}

	return nil
}

func (openstackConfig OpenstackConfig) Validate() error {
	// do validation here

	return nil
}

func NewConfigFromPath(filesystem fs.FS, path string) (CpiConfig, error) {
	var config CpiConfig

	file, err := filesystem.Open(strings.TrimPrefix(path, "/"))
	if err != nil {
		return config, fmt.Errorf("failed to open configuration file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return config, fmt.Errorf("failed to read configuration file: %w", err)
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, fmt.Errorf("failed to unmarshall configuration file: %s, err: %w", path, err)
	}

	err = config.Validate()
	if err != nil {
		return config, fmt.Errorf("failed to validate configuration file: %s, err: %w", path, err)
	}

	return config, nil
}
