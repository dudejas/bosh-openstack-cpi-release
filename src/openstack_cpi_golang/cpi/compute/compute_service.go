package compute

import (
	"encoding/json"
	"errors"
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
		agentID apiv1.AgentID,
		env apiv1.VMEnv,
		cpiConfig config.CpiConfig,
	) (*servers.Server, error)

	DeleteServer(
		vmcid string,
		cpiConfig config.CpiConfig,
	) error

	SetMetadata(
		server servers.Server,
		tags properties.ServerTags,
	) error
}

type computeService struct {
	serviceClients             utils.ServiceClients
	computeFacade              ComputeFacade
	flavorResolver             FlavorResolver
	volumeConfigurator         VolumeConfigurator
	availabilityZoneProvider   AvailabilityZoneProvider
	loadbalancerServiceBuilder loadbalancer.LoadbalancerServiceBuilder
	logger                     utils.Logger
}

func NewComputeService(
	serviceClients utils.ServiceClients,
	computeFacade ComputeFacade,
	flavorResolver FlavorResolver,
	volumeConfigurator VolumeConfigurator,
	availabilityZoneProvider AvailabilityZoneProvider,
	loadbalancerServiceBuilder loadbalancer.LoadbalancerServiceBuilder,
	logger utils.Logger,
) computeService {
	return computeService{
		serviceClients:             serviceClients,
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
	agentID apiv1.AgentID,
	env apiv1.VMEnv,
	cpiConfig config.CpiConfig,
) (*servers.Server, error) {
	openstackConfig := cpiConfig.Cloud.Properties.Openstack

	flavor, err := c.flavorResolver.ResolveFlavorForInstanceType(cloudProps.InstanceType)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve flavor of instance type '%s': %w", cloudProps.InstanceType, err)
	}

	keyname, err := c.getKeyPairName(cloudProps, openstackConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve keypair: %w", err)
	}

	blockDevices, err := c.volumeConfigurator.ConfigureVolumes(stemcellCID.AsString(), openstackConfig, cloudProps, flavor)
	if err != nil {
		return nil, fmt.Errorf("failed to configure volumes: %w", err)
	}

	vmName := c.getVMName()

	userData, err := c.createServerUserData(networkConfig, cpiConfig, vmName, flavor, agentID, env)
	if err != nil {
		fmt.Errorf("failed to create user data: %w", err)
	}

	userDataJson, err := json.Marshal(userData)
	if err != nil {
		fmt.Errorf("failed to marshal user data: %w", err)
	}

	var server *servers.Server
	availabilityZones := c.availabilityZoneProvider.GetAvailabilityZones(cloudProps)
	for _, availabilityZone := range availabilityZones {
		createOpts := c.getServerCreateOpts(vmName, availabilityZone, stemcellCID, networkConfig, flavor, keyname, blockDevices, userDataJson)

		server, err = c.computeFacade.CreateServer(c.serviceClients.ServiceClient, createOpts)
		if err != nil {
			if availabilityZone == availabilityZones[len(availabilityZones)-1] {
				return nil, fmt.Errorf("failed to create server in availability zone '%s': %w", availabilityZone, err)
			}
			c.logger.Warn("failed to create server in availability zone '%s': %v, "+
				"retrying in a different availability zone", availabilityZone, err)

			continue
		}

		server, err = c.waitForServerToBecomeActive(server.ID, time.Duration(openstackConfig.StateTimeOut)*time.Second)
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
	cpiConfig config.CpiConfig,
) error {
	var errDefault404 gophercloud.ErrDefault404

	_, err := c.computeFacade.GetServer(c.serviceClients.RetryableServiceClient, serverID)
	if err != nil {
		if errors.As(err, &errDefault404) {
			c.logger.Info("compute_service", fmt.Sprintf("SKIPPING: Server deletion with id '%s' is not found", serverID))
			return nil
		}
		return fmt.Errorf("failed to retrieve server information: %w", err)
	}

	serverMetadata, err := c.computeFacade.GetServerMetadata(c.serviceClients.RetryableServiceClient, serverID)
	if err != nil {
		if errors.As(err, &errDefault404) {
			c.logger.Info("compute_service", fmt.Sprintf("SKIPPING: Metadata retrieval for server with id '%s' is not found", serverID))
			serverMetadata = map[string]string{}
		} else {
			return fmt.Errorf("failed to retrieve server metadata: %w", err)
		}
	}

	if len(serverMetadata) > 0 {
		loadbalancerService, err := c.loadbalancerServiceBuilder.Build()
		if err != nil {
			return fmt.Errorf("failed to create loadbalancer service: %w", err)
		}

		for key, value := range serverMetadata {
			if strings.HasPrefix(key, "lbaas_pool_") {
				parts := strings.Split(value, "/")
				err = loadbalancerService.DeletePoolMember(parts[0], parts[1])
				if err != nil {
					if errors.As(err, &errDefault404) {
						c.logger.Info("compute_service", fmt.Sprintf("SKIPPING: pool member deletion with id '%s' in pool '%s' is not found", parts[1], parts[0]))
						continue
					} else {
						return fmt.Errorf("failed to delete pool member: %w", err)
					}
				}
				c.logger.Info("compute_service", fmt.Sprintf("Deleted pool member with id '%s' from pool '%s'", parts[1], parts[0]))
			}
		}
	}

	err = c.computeFacade.DeleteServer(c.serviceClients.RetryableServiceClient, serverID)
	if err != nil && !errors.As(err, &errDefault404) {
		return fmt.Errorf("failed to delete server: %w", err)
	}

	timeout := time.Duration(cpiConfig.Cloud.Properties.Openstack.StateTimeOut) * time.Second
	err = c.waitForServerToBecomeDeleted(serverID, timeout)
	if err != nil {
		return fmt.Errorf("failed while waiting on the server deletion: %w", err)
	}

	c.logger.Info("compute_service", fmt.Sprintf("Deleted server with id '%s'", serverID))

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

		_, err := c.computeFacade.SetServerMetadata(c.serviceClients.ServiceClient, server.ID, metadatumOpts)
		if err != nil {
			return fmt.Errorf("failed to set metadata: %w", err)
		}
	}

	return nil
}

func (c computeService) createServerUserData(
	networkConfig properties.NetworkConfig,
	cpiConfig config.CpiConfig,
	vmName string,
	flavor flavors.Flavor,
	agentID apiv1.AgentID,
	env apiv1.VMEnv,
) (properties.UserData, error) {
	userDataNetwork := map[string]properties.UserdataNetwork{}
	for _, network := range networkConfig.AllNetworks() {

		userdataNetwork := properties.UserdataNetwork{
			Default:    network.Default,
			DNS:        network.DNS,
			IP:         network.IP,
			Gateway:    network.Gateway,
			Netmask:    network.Netmask,
			Type:       network.Type,
			CloudProps: network.CloudProps,
			Mac:        network.Mac,
		}

		if network.Type != "vip" {
			userdataNetwork.UseDHCP = &cpiConfig.Cloud.Properties.Openstack.UseDHCP
		}

		userDataNetwork[network.Key] = userdataNetwork
	}

	environment, err := env.MarshalJSON()
	if err != nil {
		return properties.UserData{}, fmt.Errorf("failed to marshal environment")
	}

	return properties.NewUserDataBuilder().
		WithServer(properties.Server{Name: vmName}).
		WithNetworks(userDataNetwork).
		WithVM(properties.VM{Name: vmName}).
		WithNetworks(userDataNetwork).
		WithEphemeralDiskSize(flavor.Disk).
		WithAgentID(agentID).
		WithEnvironment(environment).
		WithConfig(cpiConfig).
		Build(), nil
}

func (c computeService) getServerCreateOpts(
	vmName string,
	availabilityZone string,
	stemcellCID apiv1.StemcellCID,
	networkConfig properties.NetworkConfig,
	flavor flavors.Flavor, keyname string,
	blockDevices []bootfromvolume.BlockDevice,
	userDataJson []byte,
) servers.CreateOptsBuilder {

	var createOpts servers.CreateOptsBuilder
	createOpts = servers.CreateOpts{
		Name:             vmName,
		ImageRef:         stemcellCID.AsString(),
		Networks:         c.getServerNetworks(networkConfig),
		AvailabilityZone: availabilityZone,
		FlavorRef:        flavor.ID,
		UserData:         userDataJson,

		//Security groups are set for dynamic networks here.
		//For manual networks, security groups are set on the port.
		SecurityGroups: networkConfig.SecurityGroups,
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

	keypair, err := c.computeFacade.GetOSKeyPair(c.serviceClients.RetryableServiceClient, keyPairName, keypairs.GetOpts{})
	if err != nil {
		return "", fmt.Errorf("failed to retrieve '%s': %w", keyPairName, err)
	}

	return keypair.Name, nil
}

func (c computeService) getServerNetworks(networkConfig properties.NetworkConfig) []servers.Network {
	var serverNetworks []servers.Network
	for _, network := range networkConfig.ManualNetworks {
		serverNetworks = append(serverNetworks, servers.Network{UUID: network.CloudProps.NetID, Port: network.Port.ID})
	}

	dynamicNetwork := networkConfig.DynamicNetwork
	if dynamicNetwork != nil {
		serverNetworks = append(serverNetworks, servers.Network{UUID: dynamicNetwork.CloudProps.NetID})
	}
	return serverNetworks
}

func (c computeService) getVMName() string {
	return "vm-" + uuid.New().String()
}

func (c computeService) waitForServerToBecomeActive(serverID string, timeout time.Duration) (*servers.Server, error) {
	timeoutTimer := time.NewTimer(timeout)

	for {
		select {
		case <-timeoutTimer.C:
			return nil, fmt.Errorf("timeout while waiting for server to become active")
		default:
			server, err := c.computeFacade.GetServer(c.serviceClients.RetryableServiceClient, serverID)
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
	var errDefault404 gophercloud.ErrDefault404
	timeoutTimer := time.NewTimer(timeout)

	for {
		select {
		case <-timeoutTimer.C:
			return fmt.Errorf("timeout while waiting for server to become deleted")
		default:
			server, err := c.computeFacade.GetServer(c.serviceClients.RetryableServiceClient, serverID)
			if err != nil {
				if errors.As(err, &errDefault404) {
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
