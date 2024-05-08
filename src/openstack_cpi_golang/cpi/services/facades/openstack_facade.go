package facades

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . OpenstackFacade
type OpenstackFacade interface {
	NewImageServiceV2(client *gophercloud.ProviderClient, eo gophercloud.EndpointOpts) (*gophercloud.ServiceClient, error)

	AuthenticatedClient(options gophercloud.AuthOptions) (*gophercloud.ProviderClient, error)
}

type openstackFacade struct{}

func NewOpenstackFacade() OpenstackFacade {
	return openstackFacade{}
}

func (c openstackFacade) NewImageServiceV2(client *gophercloud.ProviderClient, endpointOpts gophercloud.EndpointOpts) (*gophercloud.ServiceClient, error) {
	return openstack.NewImageServiceV2(client, endpointOpts)
}

func (c openstackFacade) AuthenticatedClient(options gophercloud.AuthOptions) (*gophercloud.ProviderClient, error) {
	return openstack.AuthenticatedClient(options)
}
