package loadbalancer

import (
	"errors"
	"fmt"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/pools"
	"time"
)

var LoadbalancerServicePollingInterval = 10 * time.Second

//counterfeiter:generate . LoadbalancerService
type LoadbalancerService interface {
	GetPool(poolName string) (pools.Pool, error)

	CreatePoolMember(poolID string, ip string, pool properties.LoadbalancerPool, subnetID string, timeout int) (*pools.Member, error)

	DeletePoolMember(poolID string, memberID string, timeout int) error
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

func (l loadbalancerService) GetPool(poolName string) (pools.Pool, error) {
	listOpts := pools.ListOpts{
		Name: poolName,
	}

	page, err := l.loadbalancerFacade.ListPools(l.serviceClients.RetryableServiceClient, listOpts)
	if err != nil {
		return pools.Pool{}, fmt.Errorf("failed to list loadbalancer pools: %w", err)
	}

	extractedPools, err := l.loadbalancerFacade.ExtractPools(page)
	if err != nil {
		return pools.Pool{}, fmt.Errorf("failed to extract loadbalancer pool pages: %w", err)
	}

	if len(extractedPools) == 0 {
		return pools.Pool{}, fmt.Errorf("loadbalancer pool '%s' does not exist", poolName)
	}

	if len(extractedPools) > 1 {
		return pools.Pool{}, fmt.Errorf("found more than one loadbalancer pool with name '%s'. Make sure to use unique naming", poolName)
	}

	return extractedPools[0], nil
}

func (l loadbalancerService) CreatePoolMember(poolID string, ip string, loadbalancerPool properties.LoadbalancerPool, subnetID string, stateTimeOut int) (*pools.Member, error) {
	var err error
	var errDefault409 gophercloud.ErrDefault409
	var poolMember *pools.Member

	createMemberOpts := pools.CreateMemberOpts{
		Address:      ip,
		ProtocolPort: loadbalancerPool.ProtocolPort,
		SubnetID:     subnetID,
	}

	if loadbalancerPool.MonitoringPort != nil {
		createMemberOpts.MonitorPort = loadbalancerPool.MonitoringPort
	}

	timeoutDuration := time.Duration(stateTimeOut) * time.Second
	createPoolMemberTimeoutTimer := time.NewTimer(timeoutDuration)
	attempts := 0

	for {
		select {
		case <-createPoolMemberTimeoutTimer.C:
			return nil, fmt.Errorf("timeout after %v attempts while creating pool membership of IP '%s' in pool '%s'", attempts, ip, loadbalancerPool.Name)
		default:
			poolMember, err = l.createPoolMember(poolID, createMemberOpts, timeoutDuration)
			if err != nil {
				if errors.As(err, &errDefault409) {
					attempts++
					l.logger.Warn("create_vm_method", fmt.Sprintf("Changing load balancer resource failed in attempt number '%v' with error: %s", attempts, err.Error()))
				} else {
					return nil, err
				}
			}
		}
		if poolMember != nil {
			break
		}
	}

	poolMember, err = l.waitForPoolMemberToBecomeActive(poolID, poolMember.ID, timeoutDuration)
	if err != nil {
		return nil, fmt.Errorf("failed while waiting for pool member to become active: %w", err)
	}

	return poolMember, nil
}

func (l loadbalancerService) DeletePoolMember(poolID string, memberID string, timeout int) error {
	_, err := l.waitForPoolToBecomeActive(poolID, time.Duration(timeout)*time.Second)
	if err != nil {
		return fmt.Errorf("failed while waiting for pool to become active: %w", err)
	}

	err = l.loadbalancerFacade.DeletePoolMember(l.serviceClients.RetryableServiceClient, poolID, memberID)
	if err != nil {
		return fmt.Errorf("failed to delete pool member: %w", err)
	}

	return nil
}

func (l loadbalancerService) createPoolMember(poolID string, createMemberOpts pools.CreateMemberOpts, timeout time.Duration) (*pools.Member, error) {
	_, err := l.waitForPoolToBecomeActive(poolID, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed while waiting for pool to become active: %w", err)
	}

	member, err := l.loadbalancerFacade.CreatePoolMember(l.serviceClients.ServiceClient, poolID, createMemberOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool member: %w", err)
	}

	return member, nil
}

func (l loadbalancerService) waitForPoolToBecomeActive(poolID string, timeout time.Duration) (*pools.Pool, error) {
	timeoutTimer := time.NewTimer(timeout)

	for {
		select {
		case <-timeoutTimer.C:
			return nil, fmt.Errorf("timeout while waiting for pool '%s' to become active", poolID)
		default:
			pool, err := l.loadbalancerFacade.GetPool(l.serviceClients.RetryableServiceClient, poolID)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve pool '%s': %w", poolID, err)
			}

			switch pool.ProvisioningStatus {
			case "ACTIVE":
				return pool, nil
			case "ERROR":
				return nil, fmt.Errorf("pool status ended up in ERROR state")
			default:
				time.Sleep(LoadbalancerServicePollingInterval)
				continue
			}
		}
	}
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
