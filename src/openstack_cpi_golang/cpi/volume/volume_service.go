package volume

import (
	"fmt"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	"github.com/google/uuid"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"time"
)

var VolumeServicePollingInterval = 10 * time.Second

//counterfeiter:generate . VolumeService
type VolumeService interface {
	CreateVolume(
		size int,
		cloudProps properties.CreateDisk,
		az string,
	) (*volumes.Volume, error)
	WaitForVolumeToBecomeAvailable(
		volumeID string,
		timeout time.Duration,
	) (*volumes.Volume, error)
}

type volumeService struct {
	volumeFacade   VolumeFacade
	serviceClients utils.ServiceClients
}

func NewVolumeService(serviceClients utils.ServiceClients, volumeFacade VolumeFacade) volumeService {
	return volumeService{
		volumeFacade:   volumeFacade,
		serviceClients: serviceClients,
	}
}

func (v volumeService) CreateVolume(
	size int,
	cloudProps properties.CreateDisk,
	az string,
) (*volumes.Volume, error) {
	volumeType := cloudProps.VolumeType

	uuid, _ := uuid.NewRandom()
	name := fmt.Sprintf("volume-%s", uuid)
	createOpts := v.getVolumeCreateOpts(size, az, volumeType, name)
	volume, err := v.volumeFacade.CreateDisk(v.serviceClients.ServiceClient, createOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create volume: %w", err)
	}

	return volume, nil
}

func (v volumeService) WaitForVolumeToBecomeAvailable(volumeID string, timeout time.Duration) (*volumes.Volume, error) {
	timeoutTimer := time.NewTimer(timeout)

	for {
		select {
		case <-timeoutTimer.C:
			return nil, fmt.Errorf("timeout while waiting for volume to become available")
		default:
			volume, err := v.getVolume(volumeID)
			if err != nil {
				return nil, err
			}

			switch volume.Status {
			case "available":
				return volume, nil
			case "error":
				return nil, fmt.Errorf("volume became error state while waiting to become available")
			}

			time.Sleep(VolumeServicePollingInterval)
		}
	}
}

func (v volumeService) getVolume(volumeID string) (*volumes.Volume, error) {
	return v.volumeFacade.GetVolume(v.serviceClients.RetryableServiceClient, volumeID)
}

func (v volumeService) getVolumeCreateOpts(size int, availabilityZone string, volumeType string, name string) volumes.CreateOptsBuilder {
	var createOpts volumes.CreateOptsBuilder
	createOpts = volumes.CreateOpts{
		Size:             size,
		AvailabilityZone: availabilityZone,
		VolumeType:       volumeType,
		Name:             name,
	}
	return createOpts
}
