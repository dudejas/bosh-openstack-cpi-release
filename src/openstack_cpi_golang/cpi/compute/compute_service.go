package compute

import (
	"fmt"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	"github.com/google/uuid"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/bootfromvolume"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
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
	serviceClient      *gophercloud.ServiceClient
	computeFacade      ComputeFacade
	flavorResolver     FlavorResolver
	volumeConfigurator VolumeConfigurator
	logger             utils.Logger
}

func NewComputeService(
	serviceClient *gophercloud.ServiceClient,
	computeFacade ComputeFacade,
	flavorResolver FlavorResolver,
	volumeConfigurator VolumeConfigurator,
	logger utils.Logger,
) computeService {
	return computeService{
		serviceClient:      serviceClient,
		computeFacade:      computeFacade,
		flavorResolver:     flavorResolver,
		volumeConfigurator: volumeConfigurator,
		logger:             logger,
	}
}

func (c computeService) CreateServer(
	stemcellCID apiv1.StemcellCID,
	cloudProps properties.CreateVM,
	networkConfig properties.NetworkConfig,
	config config.OpenstackConfig,
) (string, error) {
	flavor, err := c.flavorResolver.ResolveFlavorForInstanceType(cloudProps.InstanceType)
	if err != nil {
		return "", fmt.Errorf("failed to resolve flavor of instance type '%s': %w", cloudProps.InstanceType, err)
	}

	keyname, err := c.getKeyPairName(cloudProps, config)
	if err != nil {
		return "", fmt.Errorf("failed to resolve keypair: %w", err)
	}

	blockDevices, err := c.volumeConfigurator.ConfigureVolumes(stemcellCID.AsString(), config, cloudProps, flavor)
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
