package network

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/pagination"
)

//counterfeiter:generate . NetworkingFacade
type NetworkingFacade interface {
	ListFloatingIps(serviceClient *gophercloud.ServiceClient, opts floatingips.ListOpts) (pagination.Page, error)

	ExtractFloatingIPs(r pagination.Page) ([]floatingips.FloatingIP, error)

	UpdateFloatingIP(serviceClient *gophercloud.ServiceClient, floatingIpId string, updateOpts floatingips.UpdateOpts) (*floatingips.FloatingIP, error)

	ListPorts(client *gophercloud.ServiceClient, opts ports.ListOpts) (pagination.Page, error)

	ExtractPorts(r pagination.Page) ([]ports.Port, error)

	GetSecurityGroups(serviceClient *gophercloud.ServiceClient, id string) (*groups.SecGroup, error)

	ListSecurityGroups(serviceClient *gophercloud.ServiceClient, opts groups.ListOpts) (pagination.Page, error)

	ExtractSecurityGroups(pages pagination.Page) ([]groups.SecGroup, error)
}

type networkingFacade struct{}

func NewNetworkingFacade() NetworkingFacade {
	return networkingFacade{}
}

func (n networkingFacade) ListFloatingIps(serviceClient *gophercloud.ServiceClient, opts floatingips.ListOpts) (pagination.Page, error) {
	return floatingips.List(serviceClient, opts).AllPages()
}

func (n networkingFacade) ExtractFloatingIPs(pages pagination.Page) ([]floatingips.FloatingIP, error) {
	return floatingips.ExtractFloatingIPs(pages)
}

func (n networkingFacade) UpdateFloatingIP(serviceClient *gophercloud.ServiceClient, floatingIpId string, updateOpts floatingips.UpdateOpts) (*floatingips.FloatingIP, error) {
	return floatingips.Update(serviceClient, floatingIpId, updateOpts).Extract()
}

func (n networkingFacade) ListPorts(serviceClient *gophercloud.ServiceClient, opts ports.ListOpts) (pagination.Page, error) {
	return ports.List(serviceClient, opts).AllPages()
}

func (n networkingFacade) ExtractPorts(pages pagination.Page) ([]ports.Port, error) {
	return ports.ExtractPorts(pages)
}

func (n networkingFacade) GetSecurityGroups(serviceClient *gophercloud.ServiceClient, id string) (*groups.SecGroup, error) {
	return groups.Get(serviceClient, id).Extract()
}

func (n networkingFacade) ListSecurityGroups(serviceClient *gophercloud.ServiceClient, opts groups.ListOpts) (pagination.Page, error) {
	return groups.List(serviceClient, opts).AllPages()
}

func (n networkingFacade) ExtractSecurityGroups(pages pagination.Page) ([]groups.SecGroup, error) {
	return groups.ExtractGroups(pages)
}
