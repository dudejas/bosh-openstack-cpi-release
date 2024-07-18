package loadbalancer

import (
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/pools"
	"github.com/gophercloud/gophercloud/pagination"
)

//counterfeiter:generate . LoadbalancerFacade
type LoadbalancerFacade interface {
	GetPool(client utils.RetryableServiceClient, poolID string) (*pools.Pool, error)

	ListPools(client utils.RetryableServiceClient, listOpts pools.ListOpts) (pagination.Page, error)

	ExtractPools(allPages pagination.Page) ([]pools.Pool, error)

	CreatePoolMember(client utils.ServiceClient, poolID string, opts pools.CreateMemberOpts) (*pools.Member, error)

	GetPoolMember(client utils.RetryableServiceClient, poolID string, memberID string) (*pools.Member, error)

	DeletePoolMember(client utils.RetryableServiceClient, poolID string, memberID string) error
}

type loadbalancerFacade struct {
}

func NewLoadbalancerFacade() loadbalancerFacade {
	return loadbalancerFacade{}
}

func (l loadbalancerFacade) GetPool(client utils.RetryableServiceClient, poolID string) (*pools.Pool, error) {
	return pools.Get(client, poolID).Extract()
}

func (l loadbalancerFacade) ListPools(client utils.RetryableServiceClient, listOpts pools.ListOpts) (pagination.Page, error) {
	return pools.List(client, listOpts).AllPages()
}

func (l loadbalancerFacade) ExtractPools(allPages pagination.Page) ([]pools.Pool, error) {
	return pools.ExtractPools(allPages)
}

func (l loadbalancerFacade) CreatePoolMember(client utils.ServiceClient, poolID string, opts pools.CreateMemberOpts) (*pools.Member, error) {
	return pools.CreateMember(client, poolID, opts).Extract()
}

func (l loadbalancerFacade) GetPoolMember(client utils.RetryableServiceClient, poolID string, memberID string) (*pools.Member, error) {
	return pools.GetMember(client, poolID, memberID).Extract()
}

func (l loadbalancerFacade) DeletePoolMember(client utils.RetryableServiceClient, poolID string, memberID string) error {
	return pools.DeleteMember(client, poolID, memberID).ExtractErr()
}
