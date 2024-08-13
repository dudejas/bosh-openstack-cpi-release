package volume

import (
	"errors"
	"fmt"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	"github.com/google/uuid"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/volumeactions"
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
	WaitForVolumeToBecomeStatus(
		volumeID string,
		timeout time.Duration,
		status string,
	) error
	GetVolume(volumeID string) (*volumes.Volume, error)
	DeleteVolume(volumeId string) error
	ExtendVolumeSize(
		volumeID string,
		size int,
	) error
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
	volume, err := v.volumeFacade.CreateVolume(v.serviceClients.ServiceClient, createOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create volume: %w", err)
	}

	return volume, nil
}

func (v volumeService) WaitForVolumeToBecomeStatus(volumeID string, timeout time.Duration, status string) error {
	timeoutTimer := time.NewTimer(timeout)
	var errDefault404 gophercloud.ErrDefault404

	for {
		select {
		case <-timeoutTimer.C:
			return fmt.Errorf("timeout while waiting for volume to become %s", status)
		default:
			volume, err := v.GetVolume(volumeID)
			if err != nil {
				if errors.As(err, &errDefault404) && status == "deleted" {
					return nil
				}
				return err
			}

			switch volume.Status {
			case status:
				return nil
			case "error":
				return fmt.Errorf("volume became error state while waiting to become %s", status)
			}

			time.Sleep(VolumeServicePollingInterval)
		}
	}
}

func (v volumeService) GetVolume(volumeID string) (*volumes.Volume, error) {
	volume, err := v.volumeFacade.GetVolume(v.serviceClients.RetryableServiceClient, volumeID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve volume information: %w", err)
	}
	return volume, nil
}

func (v volumeService) ExtendVolumeSize(volumeID string, size int) error {
	var extendOpts volumeactions.ExtendSizeOptsBuilder
	extendOpts = volumeactions.ExtendSizeOpts{
		NewSize: size,
	}

	err := v.volumeFacade.ExtendVolumeSize(v.serviceClients.ServiceClient, volumeID, extendOpts)
	if err != nil {
		return fmt.Errorf("failed to extend volume size: %w", err)
	}
	return nil
}

func (v volumeService) DeleteVolume(volumeID string) error {
	deleteOpts := v.getVolumeDeleteOpts()

	err := v.volumeFacade.DeleteVolume(v.serviceClients.RetryableServiceClient, volumeID, deleteOpts)
	if err != nil {
		return fmt.Errorf("failed to delete volume: %w", err)
	}
	return nil
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

func (v volumeService) getVolumeDeleteOpts() volumes.DeleteOptsBuilder {
	var deleteOpts volumes.DeleteOptsBuilder
	deleteOpts = volumes.DeleteOpts{}
	return deleteOpts
}
