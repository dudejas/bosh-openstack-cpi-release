package services

import (
	"fmt"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services/facades"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	"github.com/google/uuid"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/bootfromvolume"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"time"
)

var ComputeServicePollingInterval = 10 * time.Second

//counterfeiter:generate . ComputeService
type ComputeService interface {
	CreateServer(
		stemcellCID apiv1.StemcellCID,
		cloudProps properties.CreateVM,
		networkConfig properties.NetworkConfig,
		config config.OpenstackConfig,
	) (string, error)
}

type computeService struct {
	serviceClient *gophercloud.ServiceClient
	computeFacade facades.ComputeFacade
	logger        utils.Logger
}

func NewComputeService(
	serviceClient *gophercloud.ServiceClient,
	computeFacade facades.ComputeFacade,
	logger utils.Logger,
) computeService {
	return computeService{
		serviceClient: serviceClient,
		computeFacade: computeFacade,
		logger:        logger,
	}
}

func (c computeService) CreateServer(
	stemcellCID apiv1.StemcellCID,
	cloudProps properties.CreateVM,
	networkConfig properties.NetworkConfig,
	config config.OpenstackConfig,
) (string, error) {
	flavor, err := c.getFlavorForInstanceType(cloudProps.InstanceType, c.serviceClient, c.computeFacade)
	if err != nil {
		return "", fmt.Errorf("failed to get flavor of instance type: %w", err)
	}

	keyname, err := c.getKeyPairName(cloudProps, config)
	if err != nil {
		return "", fmt.Errorf("failed to resolve keypair: %w", err)
	}

	blockDevices, err := c.configureVolumes(stemcellCID.AsString(), config, cloudProps, flavor)
	if err != nil {
		return "", fmt.Errorf("failed to configure volumes: %w", err)
	}

	var createOpts servers.CreateOptsBuilder
	createOpts = servers.CreateOpts{
		Name:             "vm-" + uuid.New().String(),
		ImageRef:         stemcellCID.AsString(),
		Networks:         c.getServerNetworks(networkConfig),
		SecurityGroups:   networkConfig.SecurityGroups,
		AvailabilityZone: cloudProps.AvailabilityZone,
		FlavorRef:        flavor.ID,
	}

	createOpts = keypairs.CreateOptsExt{
		CreateOptsBuilder: createOpts,
		KeyName:           keyname,
	}

	if len(blockDevices) > 0 {
		createOpts = bootfromvolume.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			BlockDevice:       blockDevices,
		}
	}

	server, err := c.computeFacade.CreateServer(c.serviceClient, createOpts)
	if err != nil {
		return "", fmt.Errorf("failed to create server: %w", err)
	}

	err = c.waitForServerToBecomeActive(server.ID, time.Duration(config.StateTimeOut)*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed while waiting on the server creation: %w", err)
	}

	return server.ID, nil
}

func (c computeService) getKeyPairName(cloudProps properties.CreateVM, openstackConfig config.OpenstackConfig) (string, error) {
	var keyPairName string

	if cloudProps.KeyName != "" {
		keyPairName = cloudProps.KeyName
	} else {
		keyPairName = openstackConfig.DefaultKeyName
	}

	if keyPairName == "" {
		return "", fmt.Errorf("key pair name undefined")
	}

	keypair, err := c.computeFacade.GetOSKeyPair(c.serviceClient, keyPairName, keypairs.GetOpts{})
	if err != nil {
		return "", fmt.Errorf("failed to retrieve '%s': %w", keyPairName, err)
	}

	return keypair.Name, nil
}

func (c computeService) configureVolumes(imageID string, openstackConfig config.OpenstackConfig, cloudProperties properties.CreateVM, flavor flavors.Flavor) ([]bootfromvolume.BlockDevice, error) {
	bootVolumeSize, err := c.select_boot_volume_size(flavor, cloudProperties)
	if err != nil {
		return []bootfromvolume.BlockDevice{}, fmt.Errorf("failed to get volume size: %w", err)
	}

	if !c.bootFromVolume(openstackConfig, cloudProperties) {
		return []bootfromvolume.BlockDevice{}, nil
	}

	return []bootfromvolume.BlockDevice{{
		UUID:                imageID,
		SourceType:          bootfromvolume.SourceImage,
		DestinationType:     bootfromvolume.DestinationVolume,
		VolumeSize:          bootVolumeSize,
		BootIndex:           0,
		DeleteOnTermination: true,
	}}, nil
}

func (c computeService) bootFromVolume(openstackConfig config.OpenstackConfig, cloudProperties properties.CreateVM) bool {
	if cloudProperties.BootFromVolume == nil {
		return openstackConfig.BootFromVolume
	}

	return *cloudProperties.BootFromVolume
}

func (c computeService) select_boot_volume_size(flavor flavors.Flavor, cloudProperties properties.CreateVM) (int, error) {
	rootDiskSize := cloudProperties.RootDisk.Size
	if rootDiskSize == 0 {
		if flavor.Disk == 0 {
			return 0, fmt.Errorf("flavor '%s' has a root disk size of 0. Either pick a different flavor or define root_disk.size in your VM cloud_properties", flavor.ID)
		}
		return flavor.Disk, nil
	}

	return rootDiskSize, nil
}

func (c computeService) getServerNetworks(networkConfig properties.NetworkConfig) []servers.Network {
	var serverNetworks []servers.Network
	for _, network := range networkConfig.ManualNetworks {
		serverNetworks = append(serverNetworks, servers.Network{UUID: network.CloudProps.NetID, FixedIP: network.IP})
	}

	dynamicNetwork := networkConfig.DynamicNetwork
	if dynamicNetwork != nil {
		serverNetworks = append(serverNetworks, servers.Network{UUID: dynamicNetwork.CloudProps.NetID})
	}
	return serverNetworks
}

func (c computeService) waitForServerToBecomeActive(serverID string, timeout time.Duration) error {
	timeoutTimer := time.NewTimer(timeout)

	for {
		select {
		case <-timeoutTimer.C:
			return fmt.Errorf("timeout while waiting for server to become active")
		default:
			server, err := c.computeFacade.GetServer(c.serviceClient, serverID)
			if err != nil {
				return fmt.Errorf("failed to retrieve server information: %w", err)
			}

			switch server.Status {
			case "ACTIVE":
				return nil
			case "ERROR":
				return fmt.Errorf("server became ERROR state while waiting to become ACTIVE")
			case "DELETED":
				return fmt.Errorf("server became DELETED state while waiting to become ACTIVE")
			}

			time.Sleep(ComputeServicePollingInterval)
		}
	}
}

func (c computeService) getFlavorForInstanceType(
	flavorName string,
	serviceClient *gophercloud.ServiceClient,
	computeFacade facades.ComputeFacade,
) (flavors.Flavor, error) {
	flavorPages, err := computeFacade.ListFlavors(serviceClient, flavors.ListOpts{})
	if err != nil {
		return flavors.Flavor{}, fmt.Errorf("failed to list flavors: %w", err)
	}

	allFlavors, err := computeFacade.ExtractFlavors(flavorPages)
	if err != nil {
		return flavors.Flavor{}, fmt.Errorf("failed to extract flavors: %w", err)
	}

	var flavor *flavors.Flavor
	for _, singleFlavor := range allFlavors {
		if singleFlavor.Name == flavorName {
			flavor = &singleFlavor
			break
		}
	}

	if flavor == nil {
		return flavors.Flavor{}, fmt.Errorf("flavor '%s' not found", flavorName)
	}

	if flavor.Ephemeral > 0 {
		// Ephemeral disk size should be at least the double of the vm total memory size, as agent will need:
		// - vm total memory size for swapon,
		// - the rest for /var/vcap/data
		minEphemeralSize := (flavor.RAM / 1024) * 2
		if flavor.Ephemeral < minEphemeralSize {
			return flavors.Flavor{}, fmt.Errorf("flavor %s should have at least %dGb of ephemeral disk", flavorName, minEphemeralSize)
		}
	}

	return *flavor, nil
}
