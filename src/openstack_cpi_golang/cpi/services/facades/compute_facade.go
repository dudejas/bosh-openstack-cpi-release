package facades

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/pagination"
)

//counterfeiter:generate . ComputeFacade
type ComputeFacade interface {
	CreateServer(client *gophercloud.ServiceClient, opts servers.CreateOptsBuilder) (*servers.Server, error)

	GetServer(client *gophercloud.ServiceClient, serverID string) (*servers.Server, error)

	ListFlavors(client *gophercloud.ServiceClient, opts flavors.ListOptsBuilder) pagination.Pager
}

type computeFacade struct {
}

func NewComputeFacade() computeFacade {
	return computeFacade{}
}

func (c computeFacade) CreateServer(client *gophercloud.ServiceClient, opts servers.CreateOptsBuilder) (*servers.Server, error) {
	return servers.Create(client, opts).Extract()
}

func (c computeFacade) GetServer(client *gophercloud.ServiceClient, serverID string) (*servers.Server, error) {
	return servers.Get(client, serverID).Extract()
}
func (c computeFacade) ListFlavors(client *gophercloud.ServiceClient, opts flavors.ListOptsBuilder) pagination.Pager {
	return flavors.ListDetail(client, opts)
}
