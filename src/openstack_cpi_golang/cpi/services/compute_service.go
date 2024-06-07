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
	flavorRef, err := c.getInstanceTypeFlavorID(cloudProps.InstanceType, c.serviceClient, c.computeFacade)
	if err != nil {
		return "", fmt.Errorf("failed to get flavor of instance type: %w", err)
	}

	serverCreateOpts := servers.CreateOpts{
		Name:             "vm-" + uuid.New().String(),
		ImageRef:         stemcellCID.AsString(),
		Networks:         c.getServerNetworks(networkConfig),
		SecurityGroups:   networkConfig.SecurityGroups,
		AvailabilityZone: cloudProps.AvailabilityZone,
		FlavorRef:        flavorRef,
	}

	createOpts := keypairs.CreateOptsExt{
		CreateOptsBuilder: serverCreateOpts,
		KeyName:           config.DefaultKeyName,
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

func (c computeService) getInstanceTypeFlavorID(
	flavorName string,
	serviceClient *gophercloud.ServiceClient,
	computeFacade facades.ComputeFacade,
) (string, error) {
	flavorPages, err := computeFacade.ListFlavors(serviceClient, nil)
	if err != nil {
		return "", fmt.Errorf("failed to list flavors: %w", err)
	}

	allFlavors, err := computeFacade.ExtractFlavors(flavorPages)
	if err != nil {
		return "", fmt.Errorf("failed to extract flavors: %w", err)
	}

	var flavor *flavors.Flavor
	for _, singleFlavor := range allFlavors {
		if singleFlavor.Name == flavorName {
			flavor = &singleFlavor
			break
		}
	}

	if flavor == nil {
		return "", fmt.Errorf("flavor '%s' not found", flavorName)
	}

	if flavor.Ephemeral > 0 {
		// Ephemeral disk size should be at least the double of the vm total memory size, as agent will need:
		// - vm total memory size for swapon,
		// - the rest for /var/vcap/data
		minEphemeralSize := (flavor.RAM / 1024) * 2
		if flavor.Ephemeral < minEphemeralSize {
			return "", fmt.Errorf("flavor %s should have at least %dGb of ephemeral disk", flavorName, minEphemeralSize)
		}
	}

	return flavor.ID, nil
}
