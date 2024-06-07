package services

import (
	"fmt"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services/facades"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
)

//counterfeiter:generate . NetworkService
type NetworkService interface {
	ConfigureNetwork(
		instanceId string,
		networkConfig properties.NetworkConfig,
	) error

	ResolveSecurityGroups(
		securityGroupIDsAndNames []string,
	) ([]string, error)
}

type networkService struct {
	serviceClient    *gophercloud.ServiceClient
	networkingFacade facades.NetworkingFacade
	logger           utils.Logger
}

func NewNetworkService(
	serviceClient *gophercloud.ServiceClient,
	networkingFacade facades.NetworkingFacade,
	logger utils.Logger,
) networkService {
	return networkService{
		serviceClient:    serviceClient,
		networkingFacade: networkingFacade,
		logger:           logger,
	}
}

func (c networkService) ConfigureNetwork(
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

func (c networkService) ResolveSecurityGroups(securityGroupIDsAndNames []string) ([]string, error) {
	var securityGroupIds []string
	var resolvedSecurityGroup *groups.SecGroup
	var err error

	for _, securityGroup := range securityGroupIDsAndNames {
		resolvedSecurityGroup, err = c.resolveSecurityGroupById(securityGroup)
		if err != nil {
			return []string{}, fmt.Errorf("failed to get security group '%s' by id: %w", securityGroup, err)
		}

		if resolvedSecurityGroup != nil {
			securityGroupIds = append(securityGroupIds, resolvedSecurityGroup.ID)
			continue
		} else {
			resolvedSecurityGroup, err = c.resolveSecurityGroupByName(securityGroup)
			if err != nil {
				return []string{}, fmt.Errorf("failed to get security group '%s' by name: %w", securityGroup, err)
			}

			securityGroupIds = append(securityGroupIds, resolvedSecurityGroup.ID)
			continue
		}

	}
	return securityGroupIds, nil
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

func (c networkService) resolveSecurityGroupById(securityGroupID string) (*groups.SecGroup, error) {
	return c.networkingFacade.GetSecurityGroups(c.serviceClient, securityGroupID)
}

func (c networkService) resolveSecurityGroupByName(securityGroupName string) (*groups.SecGroup, error) {
	listOpts := groups.ListOpts{
		Name: securityGroupName,
	}

	allPages, err := c.networkingFacade.ListSecurityGroups(c.serviceClient, listOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list security groups: %w", err)
	}

	allSecurityGroups, err := c.networkingFacade.ExtractSecurityGroups(allPages)
	if err != nil {
		return nil, fmt.Errorf("failed to extract security groups: %w", err)
	}

	if len(allSecurityGroups) == 0 {
		return nil, fmt.Errorf("security group '%s' could not be found", securityGroupName)
	}

	return &allSecurityGroups[0], nil
}
