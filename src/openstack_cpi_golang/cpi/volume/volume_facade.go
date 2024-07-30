package volume

import (
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
)

//counterfeiter:generate . VolumeFacade
type VolumeFacade interface {
	CreateVolume(client utils.ServiceClient, opts volumes.CreateOptsBuilder) (*volumes.Volume, error)
	GetVolume(client utils.RetryableServiceClient, volumeID string) (*volumes.Volume, error)
	DeleteVolume(client utils.RetryableServiceClient, volumeID string, opts volumes.DeleteOptsBuilder) error
}

type volumeFacade struct{}

func (v volumeFacade) CreateVolume(client utils.ServiceClient, opts volumes.CreateOptsBuilder) (*volumes.Volume, error) {
	return volumes.Create(client, opts).Extract()
}

func (v volumeFacade) GetVolume(client utils.RetryableServiceClient, volumeID string) (*volumes.Volume, error) {
	return volumes.Get(client, volumeID).Extract()
}

func (v volumeFacade) DeleteVolume(client utils.RetryableServiceClient, volumeID string, opts volumes.DeleteOptsBuilder) error {
	err := volumes.Delete(client, volumeID, opts).ExtractErr()
	if err != nil {
		return err
	}
	return nil
}

func NewVolumeFacade() volumeFacade {
	return volumeFacade{}
}
