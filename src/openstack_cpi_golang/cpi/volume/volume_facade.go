package volume

import (
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
)

//counterfeiter:generate . VolumeFacade
type VolumeFacade interface {
	CreateDisk(client utils.ServiceClient, opts volumes.CreateOptsBuilder) (*volumes.Volume, error)
	GetVolume(client utils.RetryableServiceClient, volumeID string) (*volumes.Volume, error)
}

type volumeFacade struct{}

func (v volumeFacade) CreateDisk(client utils.ServiceClient, opts volumes.CreateOptsBuilder) (*volumes.Volume, error) {
	return volumes.Create(client, opts).Extract()
}

func (v volumeFacade) GetVolume(client utils.RetryableServiceClient, volumeID string) (*volumes.Volume, error) {
	return volumes.Get(client, volumeID).Extract()
}

func NewVolumeFacade() volumeFacade {
	return volumeFacade{}
}
