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

		port, err := c.getPort(instanceId, err, networkConfig.DefaultNetwork)
		if err != nil {
			return fmt.Errorf("failed to get port: %w", err)
		}

		err = c.associateFloatingIp(c.serviceClient, floatingIp.ID, port.ID)
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

	return NewNetworkConfigBuilder(securityGroupsResolver, networks, openstackConfig, cloudProps).Build()
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

func (c networkService) getPort(instanceId string, err error, defaultNetwork properties.Network) (ports.Port, error) {
	listOpts := ports.ListOpts{
		DeviceID:  instanceId,
		NetworkID: defaultNetwork.CloudProps.NetID,
	}

	allPages, err := c.networkingFacade.ListPorts(c.serviceClient, listOpts)
	if err != nil {
		return ports.Port{}, fmt.Errorf("failed to list ports: %w", err)
	}

	allPorts, err := c.networkingFacade.ExtractPorts(allPages)
	if err != nil {
		return ports.Port{}, fmt.Errorf("failed to extract ports: %w", err)
	}

	if len(allPorts) == 0 {
		return ports.Port{}, fmt.Errorf("no port allocated by instance %s and network %s", instanceId, defaultNetwork.CloudProps.NetID)
	}

	return allPorts[0], nil
}

func (c networkService) associateFloatingIp(serviceClient *gophercloud.ServiceClient, floatingIpId string, portId string) error {
	updateOpts := floatingips.UpdateOpts{
		PortID: &portId,
	}

	_, err := c.networkingFacade.UpdateFloatingIP(serviceClient, floatingIpId, updateOpts)
	return err
}
