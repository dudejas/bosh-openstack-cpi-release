package facades

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
)

//counterfeiter:generate . ImagesFacade
type ImagesFacade interface {
	Create(client *gophercloud.ServiceClient, opts images.CreateOptsBuilder) (*images.Image, error)

	Get(client *gophercloud.ServiceClient, id string) (*images.Image, error)

	Delete(client *gophercloud.ServiceClient, id string) error
}

type imagesFacade struct{}

func NewImagesFacade() ImagesFacade {
	return imagesFacade{}
}

func (c imagesFacade) Create(serviceClient *gophercloud.ServiceClient, createOpts images.CreateOptsBuilder) (*images.Image, error) {
	return images.Create(serviceClient, createOpts).Extract()
}

func (c imagesFacade) Get(serviceClient *gophercloud.ServiceClient, id string) (*images.Image, error) {
	return images.Get(serviceClient, id).Extract()
}

func (c imagesFacade) Delete(serviceClient *gophercloud.ServiceClient, id string) error {
	return images.Delete(serviceClient, id).ExtractErr()
}
