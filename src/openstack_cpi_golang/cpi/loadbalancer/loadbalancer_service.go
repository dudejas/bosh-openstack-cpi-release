package loadbalancer

import (
	"fmt"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/pools"
	"time"
)

var LoadbalancerServicePollingInterval = 10 * time.Second

//counterfeiter:generate . LoadbalancerService
type LoadbalancerService interface {
	GetPoolID(poolName string) (string, error)

	CreatePoolMember(poolID string, ip string, pool properties.LoadbalancerPool, subnetID string, timeout int) (*pools.Member, error)

	DeletePoolMember(poolID string, memberID string) error
}

type loadbalancerService struct {
	serviceClients     utils.ServiceClients
	loadbalancerFacade LoadbalancerFacade
	logger             utils.Logger
}

func NewLoadbalancerService(
	serviceClients utils.ServiceClients,
	loadbalancerFacade LoadbalancerFacade,
	logger utils.Logger,
) loadbalancerService {
	return loadbalancerService{
		serviceClients:     serviceClients,
		loadbalancerFacade: loadbalancerFacade,
		logger:             logger,
	}
}

func (l loadbalancerService) GetPoolID(poolName string) (string, error) {
	listOpts := pools.ListOpts{
		Name: poolName,
	}

	page, err := l.loadbalancerFacade.ListPools(l.serviceClients.RetryableServiceClient, listOpts)
	if err != nil {
		return "", fmt.Errorf("failed to list loadbalancer pools: %w", err)
	}

	pools, err := l.loadbalancerFacade.ExtractPools(page)
	if err != nil {
		return "", fmt.Errorf("failed to extract loadbalancer pool pages: %w", err)
	}

	if len(pools) == 0 {
		return "", fmt.Errorf("loadbalancer pool '%s' does not exist", poolName)
	}

	if len(pools) > 1 {
		return "", fmt.Errorf("found more than one loadbalancer pool with name '%s'. Make sure to use unique naming", poolName)
	}

	return pools[0].ID, nil
}

func (l loadbalancerService) CreatePoolMember(poolID string, ip string, pool properties.LoadbalancerPool, subnetID string, timeout int) (*pools.Member, error) {
	createMemberOpts := pools.CreateMemberOpts{
		Address:      ip,
		ProtocolPort: pool.ProtocolPort,
		SubnetID:     subnetID,
	}

	if pool.MonitoringPort != nil {
		createMemberOpts.MonitorPort = pool.MonitoringPort
	}

	member, err := l.loadbalancerFacade.CreatePoolMember(l.serviceClients.ServiceClient, poolID, createMemberOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool member: %w", err)
	}

	member, err = l.waitForPoolMemberToBecomeActive(poolID, member.ID, time.Duration(timeout)*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed while waiting for pool member: %w", err)
	}

	return member, nil
}

func (l loadbalancerService) DeletePoolMember(poolID string, memberID string) error {
	err := l.loadbalancerFacade.DeletePoolMember(l.serviceClients.RetryableServiceClient, poolID, memberID)
	if err != nil {
		return fmt.Errorf("failed to delete pool member: %w", err)
	}

	return nil
}

func (l loadbalancerService) waitForPoolMemberToBecomeActive(poolID string, memberID string, timeout time.Duration) (*pools.Member, error) {
	timeoutTimer := time.NewTimer(timeout)

	for {
		select {
		case <-timeoutTimer.C:
			return nil, fmt.Errorf("timeout while waiting for pool member '%s' to become active", memberID)
		default:
			member, err := l.loadbalancerFacade.GetPoolMember(l.serviceClients.RetryableServiceClient, poolID, memberID)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve pool member '%s': %w", memberID, err)
			}

			switch member.ProvisioningStatus {
			case "ACTIVE":
				return member, nil
			case "PENDING_CREATE":
				time.Sleep(LoadbalancerServicePollingInterval)
				continue
			case "ERROR":
				return nil, fmt.Errorf("pool member creation finished with ERROR state")
			default:
				return nil, fmt.Errorf("pool member creation fails for unknown provisioning status '%s'", member.ProvisioningStatus)
			}
		}
	}
}
