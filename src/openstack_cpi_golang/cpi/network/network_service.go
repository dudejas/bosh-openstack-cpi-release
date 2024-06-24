package network

import (
	"fmt"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"net"
	"strings"
)

//counterfeiter:generate . NetworkService
type NetworkService interface {
	ConfigureVIPNetwork(
		instanceId string,
		networkConfig properties.NetworkConfig,
	) error

	GetNetworkConfiguration(
		networks apiv1.Networks,
		openstackConfig config.OpenstackConfig,
		cloudProps properties.CreateVM,
	) (properties.NetworkConfig, error)

	GetSubnetID(networkID string, ip string) (string, error)

	CreatePort(networkConfig properties.NetworkConfig, cloudProperties properties.CreateVM) (*ports.Port, error)

	GetPorts(
		instanceId string,
		defaultNetwork properties.Network,
		retryable bool,
	) ([]ports.Port, error)

	DeletePorts(
		ports []ports.Port,
	) error
}

type networkService struct {
	serviceClient    *gophercloud.ServiceClient
	networkingFacade NetworkingFacade
	logger           utils.Logger
}

func NewNetworkService(
	serviceClient *gophercloud.ServiceClient,
	networkingFacade NetworkingFacade,
	logger utils.Logger,
) networkService {
	return networkService{
		serviceClient:    serviceClient,
		networkingFacade: networkingFacade,
		logger:           logger,
	}
}

func (c networkService) ConfigureVIPNetwork(
	instanceId string,
	networkConfig properties.NetworkConfig,
) error {
	vipNetwork := networkConfig.VIPNetwork

	if vipNetwork != nil {
		floatingIp, err := c.getFloatingIp(vipNetwork)
		if err != nil {
			return fmt.Errorf("failed to get floating IP: %w", err)
		}

		ports, err := c.GetPorts(instanceId, networkConfig.DefaultNetwork, false)
		if err != nil {
			return fmt.Errorf("failed to get port: %w", err)
		}
		if len(ports) == 0 {
			return fmt.Errorf("no port allocated by instance %s and network %s", instanceId, networkConfig.DefaultNetwork.CloudProps.NetID)
		}

		err = c.associateFloatingIp(c.serviceClient, floatingIp.ID, ports[0].ID)
		if err != nil {
			return fmt.Errorf("failed to associate floating ip to port: %w", err)
		}
	}
	return nil
}

func (c networkService) GetNetworkConfiguration(
	networks apiv1.Networks,
	openstackConfig config.OpenstackConfig,
	cloudProps properties.CreateVM,
) (properties.NetworkConfig, error) {
	securityGroupsResolver := NewSecurityGroupsResolver(c.serviceClient, c.networkingFacade)

	networkProperties, err := NewNetworkConfigBuilder(securityGroupsResolver, networks, openstackConfig, cloudProps).Build()
	return networkProperties, err
}

func (c networkService) GetSubnetID(networkID string, ip string) (string, error) {
	ipAddress := net.ParseIP(ip)
	if ipAddress == nil {
		return "", fmt.Errorf("failed to parse ip address '%s'", ip)
	}

	listOpts := subnets.ListOpts{
		NetworkID: networkID,
	}

	allPages, err := c.networkingFacade.ListSubnets(c.serviceClient, listOpts)
	if err != nil {
		return "", fmt.Errorf("failed to list subnets: %w", err)
	}

	allSubnets, err := c.networkingFacade.ExtractSubnets(allPages)
	if err != nil {
		return "", fmt.Errorf("failed to extract subnets: %w", err)
	}

	if len(allSubnets) == 0 {
		return "", fmt.Errorf("no subnet found for network '%s'", networkID)
	}

	var matchingSubnets []string
	for _, subnet := range allSubnets {
		_, ipNet, err := net.ParseCIDR(subnet.CIDR)
		if ipNet == nil {
			return "", fmt.Errorf("failed to parse subnet cidr '%s': %w", subnet.CIDR, err)
		}

		if ipNet.Contains(ipAddress) {
			matchingSubnets = append(matchingSubnets, subnet.ID)
		}
	}

	if len(matchingSubnets) > 1 {
		return "", fmt.Errorf("found more than one matching subnet for the ip '%s' in '%v'", ipAddress, matchingSubnets)
	}

	return matchingSubnets[0], nil
}

func (c networkService) CreatePort(networkConfig properties.NetworkConfig, cloudProperties properties.CreateVM) (*ports.Port, error) {
	defaultNetwork := networkConfig.DefaultNetwork

	createOpts, err := c.getPortCreationNetworkOpts(defaultNetwork, cloudProperties)
	if err != nil {
		return nil, fmt.Errorf("failed create network opts: %w", err)
	}

	c.logger.Info("network-service", fmt.Sprintf("Creating port with opts '%+v'", createOpts))

	createdPort, err := c.networkingFacade.CreatePort(c.serviceClient, createOpts)
	if err != nil {
		c.logger.Warn("network-service",
			fmt.Sprintf("port creation on network '%s' for ip '%s' "+
				"failed with: %v, checking conflicting ports now.",
				defaultNetwork.CloudProps.NetID, defaultNetwork.IP, err))

		listOpts := ports.ListOpts{
			NetworkID: defaultNetwork.CloudProps.NetID,
			FixedIPs:  []ports.FixedIPOpts{{IPAddress: defaultNetwork.IP}},
		}
		page, err := c.networkingFacade.ListPorts(c.serviceClient, listOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to list Ports: %w", err)
		}

		ports, err := c.networkingFacade.ExtractPorts(page)
		if err != nil {
			return nil, fmt.Errorf("failed to extract ports: %w", err)
		}

		for _, port := range ports {
			if port.Status == "DOWN" && port.DeviceID == "" && port.DeviceOwner == "" {
				c.logger.Warn("network-service", fmt.Sprintf("port on network '%s' for ip '%s' "+
					"is already allocated but unused, deleting conflicting port now.",
					defaultNetwork.CloudProps.NetID, defaultNetwork.IP))

				err := c.networkingFacade.DeletePort(c.serviceClient, port.ID)
				if err != nil {
					return nil, fmt.Errorf("failed to delete port: %w", err)
				}
			}
		}

		createdPort, err = c.networkingFacade.CreatePort(c.serviceClient, createOpts)
		if err != nil {
			return nil, fmt.Errorf("port creation on network '%s' for ip '%s' "+
				"failed with: %w, on second attempt",
				defaultNetwork.CloudProps.NetID, defaultNetwork.IP, err)
		}
	}

	return createdPort, nil
}

func (c networkService) GetPorts(instanceId string, defaultNetwork properties.Network, retryable bool) ([]ports.Port, error) {
	serviceClient := c.serviceClient
	if retryable {
		serviceClient.RetryFunc = utils.RetryOnError(c.logger)
	}

	listOpts := ports.ListOpts{
		DeviceID: instanceId,
	}

	if defaultNetwork.CloudProps.NetID != "" {
		listOpts.NetworkID = defaultNetwork.CloudProps.NetID
	}

	allPages, err := c.networkingFacade.ListPorts(serviceClient, listOpts)
	if err != nil {
		return []ports.Port{}, fmt.Errorf("failed to list ports: %w", err)
	}

	allPorts, err := c.networkingFacade.ExtractPorts(allPages)
	if err != nil {
		return []ports.Port{}, fmt.Errorf("failed to extract ports: %w", err)
	}

	return allPorts, nil
}

func (c networkService) DeletePorts(ports []ports.Port) error {
	serviceClient := c.serviceClient
	serviceClient.RetryFunc = utils.RetryOnError(c.logger)

	for _, port := range ports {
		err := c.networkingFacade.DeletePort(serviceClient, port.ID)
		if err != nil {
			if strings.Contains(err.Error(), "Resource not found") {
				c.logger.Info("network_service", fmt.Sprintf("SKIPPING: Port deletion with id '%s' is not found", port.ID))
				return nil
			}
			return fmt.Errorf("failed to delete port: %w", err)
		}
		c.logger.Info("network_service", fmt.Sprintf("Deleted port with id '%s'", port.ID))
	}

	return nil
}

func (c networkService) getPortCreationNetworkOpts(defaultNetwork properties.Network, cloudProperties properties.CreateVM) (ports.CreateOpts, error) {

	subnetID, err := c.GetSubnetID(defaultNetwork.CloudProps.NetID, defaultNetwork.IP)
	if err != nil {
		return ports.CreateOpts{}, fmt.Errorf("failed to get subnet: %w", err)
	}

	createOpts := ports.CreateOpts{
		NetworkID: defaultNetwork.CloudProps.NetID,
		FixedIPs: []ports.IP{
			{SubnetID: subnetID, IPAddress: defaultNetwork.IP},
		},
	}

	if cloudProperties.AllowedAddressPairs != "" {
		vrrpPortExisting, err := c.isVRRPPortExisting(cloudProperties)
		if err != nil {
			return ports.CreateOpts{}, fmt.Errorf("VRRP port existence check failed: %w", err)
		}

		if !vrrpPortExisting {
			return ports.CreateOpts{}, fmt.Errorf("configured VRRP port with ip '%s' does not exist", cloudProperties.AllowedAddressPairs)
		}

		createOpts.AllowedAddressPairs = []ports.AddressPair{{IPAddress: cloudProperties.AllowedAddressPairs}}
	}
	return createOpts, nil
}

func (c networkService) isVRRPPortExisting(cloudProperties properties.CreateVM) (bool, error) {
	vrrpPortCheck := cloudProperties.VRRPPortCheck
	if vrrpPortCheck != nil && *vrrpPortCheck {
		listOpts := ports.ListOpts{
			FixedIPs: []ports.FixedIPOpts{{IPAddress: cloudProperties.AllowedAddressPairs}},
		}
		page, err := c.networkingFacade.ListPorts(c.serviceClient, listOpts)
		if err != nil {
			return false, fmt.Errorf("failed to list VRRP ports: %w", err)
		}

		ports, err := c.networkingFacade.ExtractPorts(page)
		if err != nil {
			return false, fmt.Errorf("failed to extract ports: %w", err)
		}

		if len(ports) == 0 {
			return false, nil
		}
	}
	return true, nil
}

func (c networkService) getFloatingIp(vipNetwork *properties.Network) (floatingips.FloatingIP, error) {
	listOpts := floatingips.ListOpts{
		FloatingIP: vipNetwork.IP,
	}

	allPages, err := c.networkingFacade.ListFloatingIps(c.serviceClient, listOpts)
	if err != nil {
		return floatingips.FloatingIP{}, fmt.Errorf("failed to list floating IPs: %w", err)
	}

	allFIPs, err := c.networkingFacade.ExtractFloatingIPs(allPages)
	if err != nil {
		return floatingips.FloatingIP{}, fmt.Errorf("failed to extract floating IPs: %w", err)
	}

	if len(allFIPs) == 0 {
		return floatingips.FloatingIP{}, fmt.Errorf("floating IP %s not allocated", vipNetwork.IP)
	}

	return allFIPs[0], err
}

func (c networkService) associateFloatingIp(serviceClient *gophercloud.ServiceClient, floatingIpId string, portId string) error {
	updateOpts := floatingips.UpdateOpts{
		PortID: &portId,
	}

	_, err := c.networkingFacade.UpdateFloatingIP(serviceClient, floatingIpId, updateOpts)
	return err
}
