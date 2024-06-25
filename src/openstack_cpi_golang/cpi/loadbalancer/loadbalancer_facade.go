package loadbalancer

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/pools"
	"github.com/gophercloud/gophercloud/pagination"
)

//counterfeiter:generate . LoadbalancerFacade
type LoadbalancerFacade interface {
	ListPools(client *gophercloud.ServiceClient, listOpts pools.ListOpts) (pagination.Page, error)

	ExtractPools(allPages pagination.Page) ([]pools.Pool, error)

	CreatePoolMember(client *gophercloud.ServiceClient, poolID string, opts pools.CreateMemberOpts) (*pools.Member, error)

	GetPoolMember(client *gophercloud.ServiceClient, poolID string, memberID string) (*pools.Member, error)

	//https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/pools
	DeletePoolMember(client *gophercloud.ServiceClient, poolID string, memberID string) error
}

type loadbalancerFacade struct {
}

func NewLoadbalancerFacade() loadbalancerFacade {
	return loadbalancerFacade{}
}

func (l loadbalancerFacade) ListPools(client *gophercloud.ServiceClient, listOpts pools.ListOpts) (pagination.Page, error) {
	return pools.List(client, listOpts).AllPages()
}

func (l loadbalancerFacade) ExtractPools(allPages pagination.Page) ([]pools.Pool, error) {
	return pools.ExtractPools(allPages)
}

func (l loadbalancerFacade) CreatePoolMember(client *gophercloud.ServiceClient, poolID string, opts pools.CreateMemberOpts) (*pools.Member, error) {
	return pools.CreateMember(client, poolID, opts).Extract()
}

func (l loadbalancerFacade) GetPoolMember(client *gophercloud.ServiceClient, poolID string, memberID string) (*pools.Member, error) {
	return pools.GetMember(client, poolID, memberID).Extract()
}

func (l loadbalancerFacade) DeletePoolMember(client *gophercloud.ServiceClient, poolID string, memberID string) error {
	return pools.DeleteMember(client, poolID, memberID).ExtractErr()
}
