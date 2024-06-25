package compute

import (
	"fmt"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/loadbalancer"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	"github.com/google/uuid"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/bootfromvolume"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"strings"
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
	) (*servers.Server, error)

	DeleteServer(
		vmcid string,
		config config.OpenstackConfig,
	) error

	SetMetadata(
		server servers.Server,
		tags properties.ServerTags,
	) error
}

type computeService struct {
	serviceClient              *gophercloud.ServiceClient
	computeFacade              ComputeFacade
	flavorResolver             FlavorResolver
	volumeConfigurator         VolumeConfigurator
	availabilityZoneProvider   AvailabilityZoneProvider
	loadbalancerServiceBuilder loadbalancer.LoadbalancerServiceBuilder
	logger                     utils.Logger
}

func NewComputeService(
	serviceClient *gophercloud.ServiceClient,
	computeFacade ComputeFacade,
	flavorResolver FlavorResolver,
	volumeConfigurator VolumeConfigurator,
	availabilityZoneProvider AvailabilityZoneProvider,
	loadbalancerServiceBuilder loadbalancer.LoadbalancerServiceBuilder,
	logger utils.Logger,
) computeService {
	return computeService{
		serviceClient:              serviceClient,
		computeFacade:              computeFacade,
		flavorResolver:             flavorResolver,
		volumeConfigurator:         volumeConfigurator,
		availabilityZoneProvider:   availabilityZoneProvider,
		loadbalancerServiceBuilder: loadbalancerServiceBuilder,
		logger:                     logger,
	}
}

func (c computeService) CreateServer(
	stemcellCID apiv1.StemcellCID,
	cloudProps properties.CreateVM,
	networkConfig properties.NetworkConfig,
	config config.OpenstackConfig,
) (*servers.Server, error) {
	flavor, err := c.flavorResolver.ResolveFlavorForInstanceType(cloudProps.InstanceType)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve flavor of instance type '%s': %w", cloudProps.InstanceType, err)
	}

	keyname, err := c.getKeyPairName(cloudProps, config)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve keypair: %w", err)
	}

	blockDevices, err := c.volumeConfigurator.ConfigureVolumes(stemcellCID.AsString(), config, cloudProps, flavor)
	if err != nil {
		return nil, fmt.Errorf("failed to configure volumes: %w", err)
	}

	var server *servers.Server
	availabilityZones := c.availabilityZoneProvider.GetAvailabilityZones(cloudProps)
	for _, availabilityZone := range availabilityZones {
		createOpts := c.getServerCreateOpts(availabilityZone, stemcellCID, networkConfig, flavor, keyname, blockDevices)

		server, err = c.computeFacade.CreateServer(c.serviceClient, createOpts)
		if err != nil {
			if availabilityZone == availabilityZones[len(availabilityZones)-1] {
				return nil, fmt.Errorf("failed to create server in availability zone '%s': %w", availabilityZone, err)
			}
			c.logger.Warn("failed to create server in availability zone '%s': %v, "+
				"retrying in a different availability zone", availabilityZone, err)

			continue
		}

		server, err = c.waitForServerToBecomeActive(server.ID, time.Duration(config.StateTimeOut)*time.Second)
		if err != nil {
			if availabilityZone == availabilityZones[len(availabilityZones)-1] {
				return nil, fmt.Errorf("failed while waiting on the server creation in availability zone '%s': %w", availabilityZone, err)
			}
			c.logger.Warn("failed while waiting on the server creation in availability zone '%s': %v, "+
				"retrying in a different availability zone", availabilityZone, err)
		}
	}

	return server, nil
}

func (c computeService) DeleteServer(
	serverID string,
	config config.OpenstackConfig,
) error {
	serviceClient := c.serviceClient
	serviceClient.RetryFunc = utils.RetryOnError(c.logger)

	_, err := c.computeFacade.GetServer(serviceClient, serverID)
	if err != nil {
		if strings.Contains(err.Error(), "Resource not found") {
			return nil
		}
		return fmt.Errorf("failed to retrieve server information: %w", err)
	}

	serverTags, err := c.computeFacade.GetServerTags(serviceClient, serverID)
	if err != nil {
		if strings.Contains(err.Error(), "Resource not found") {
			serverTags = []string{}
		} else {
			return fmt.Errorf("failed to retrieve server tags: %w", err)
		}
	}

	if len(serverTags) > 0 {
		loadbalancerService, err := c.loadbalancerServiceBuilder.Build()
		if err != nil {
			return fmt.Errorf("failed to create loadbalancer service: %w", err)
		}

		for _, tag := range serverTags {
			if strings.HasPrefix(tag, "lbaas_pool_") {
				tag = strings.TrimPrefix(tag, "lbaas_pool_")
				parts := strings.Split(tag, "/")
				err = loadbalancerService.DeletePoolMember(parts[0], parts[1])
				if err != nil {
					return fmt.Errorf("failed to delete pool member: %w", err)
				}
			}
		}
	}

	err = c.computeFacade.DeleteServer(serviceClient, serverID)
	if err != nil {
		return fmt.Errorf("failed to delete server: %w", err)
	}

	err = c.waitForServerToBecomeDeleted(serverID, time.Duration(config.StateTimeOut)*time.Second)
	if err != nil {
		return fmt.Errorf("failed while waiting on the server deletion: %w", err)
	}

	// deleting registry settings - Seems that it is not needed for V2
	// https://bosh.io/docs/cpi-api-v2/#reference-table-based-on-each-component-version

	return nil
}

func (c computeService) SetMetadata(server servers.Server, tags properties.ServerTags) error {

	if len(tags) > 0 {
		metadatumOpts := servers.MetadatumOpts{}
		for k, v := range tags {
			metadatumOpts[k] = v
		}

		_, err := c.computeFacade.SetServerMetadata(c.serviceClient, server.ID, metadatumOpts)
		if err != nil {
			return fmt.Errorf("failed to set metadata: %w", err)
		}
	}

	return nil
}

func (c computeService) getServerCreateOpts(
	availabilityZone string,
	stemcellCID apiv1.StemcellCID,
	networkConfig properties.NetworkConfig,
	flavor flavors.Flavor, keyname string,
	blockDevices []bootfromvolume.BlockDevice,
) servers.CreateOptsBuilder {
	var createOpts servers.CreateOptsBuilder
	createOpts = servers.CreateOpts{
		Name:             "vm-" + uuid.New().String(),
		ImageRef:         stemcellCID.AsString(),
		Networks:         c.getServerNetworks(networkConfig),
		SecurityGroups:   networkConfig.SecurityGroups,
		AvailabilityZone: availabilityZone,
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
	return createOpts
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

func (c computeService) waitForServerToBecomeActive(serverID string, timeout time.Duration) (*servers.Server, error) {
	timeoutTimer := time.NewTimer(timeout)

	for {
		select {
		case <-timeoutTimer.C:
			return nil, fmt.Errorf("timeout while waiting for server to become active")
		default:
			server, err := c.computeFacade.GetServer(c.serviceClient, serverID)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve server information: %w", err)
			}

			switch server.Status {
			case "ACTIVE":
				return server, nil
			case "ERROR":
				return nil, fmt.Errorf("server became ERROR state while waiting to become ACTIVE")
			case "DELETED":
				return nil, fmt.Errorf("server became DELETED state while waiting to become ACTIVE")
			}

			time.Sleep(ComputeServicePollingInterval)
		}
	}
}

func (c computeService) waitForServerToBecomeDeleted(serverID string, timeout time.Duration) error {
	timeoutTimer := time.NewTimer(timeout)
	serviceClient := c.serviceClient
	serviceClient.RetryFunc = utils.RetryOnError(c.logger)

	for {
		select {
		case <-timeoutTimer.C:
			return fmt.Errorf("timeout while waiting for server to become deleted")
		default:
			server, err := c.computeFacade.GetServer(serviceClient, serverID)
			if err != nil {
				if strings.Contains(err.Error(), "Resource not found") {
					return nil
				}
				return fmt.Errorf("failed to retrieve server information: %w", err)
			}

			switch server.Status {
			case "DELETED":
				return nil
			case "TERMINATED":
				return nil
			case "ERROR":
				return fmt.Errorf("server became ERROR state while waiting to become DELETED")
			}

			time.Sleep(ComputeServicePollingInterval)
		}
	}
}
