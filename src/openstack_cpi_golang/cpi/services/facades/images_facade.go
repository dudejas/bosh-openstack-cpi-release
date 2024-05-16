package facades

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
)

//counterfeiter:generate . ImagesFacade
type ImagesFacade interface {
	Create(client *gophercloud.ServiceClient, opts images.CreateOptsBuilder) (r images.CreateResult)

	Get(client *gophercloud.ServiceClient, id string) (r images.GetResult)
}

type imagesFacade struct{}

func NewImagesFacade() ImagesFacade {
	return imagesFacade{}
}

func (c imagesFacade) Create(serviceClient *gophercloud.ServiceClient, createOpts images.CreateOptsBuilder) (r images.CreateResult) {
	return images.Create(serviceClient, createOpts)
}

func (c imagesFacade) Get(serviceClient *gophercloud.ServiceClient, id string) (r images.GetResult) {
	return images.Get(serviceClient, id)
}
