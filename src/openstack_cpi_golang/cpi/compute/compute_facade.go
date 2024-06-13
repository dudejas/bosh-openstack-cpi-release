package compute

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/pagination"
)

//counterfeiter:generate . ComputeFacade
type ComputeFacade interface {
	CreateServer(client *gophercloud.ServiceClient, opts servers.CreateOptsBuilder) (*servers.Server, error)

	GetServer(client *gophercloud.ServiceClient, serverID string) (*servers.Server, error)

	ListFlavors(client *gophercloud.ServiceClient, opts flavors.ListOpts) (pagination.Page, error)

	ExtractFlavors(page pagination.Page) ([]flavors.Flavor, error)

	GetOSKeyPair(client *gophercloud.ServiceClient, keyPairName string, ops keypairs.GetOpts) (*keypairs.KeyPair, error)
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

func (c computeFacade) ListFlavors(client *gophercloud.ServiceClient, opts flavors.ListOpts) (pagination.Page, error) {
	return flavors.ListDetail(client, opts).AllPages()
}

func (c computeFacade) ExtractFlavors(page pagination.Page) ([]flavors.Flavor, error) {
	return flavors.ExtractFlavors(page)
}

func (c computeFacade) GetOSKeyPair(client *gophercloud.ServiceClient, keyPairName string, opts keypairs.GetOpts) (*keypairs.KeyPair, error) {
	return keypairs.Get(client, keyPairName, opts).Extract()
}
