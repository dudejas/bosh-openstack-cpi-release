package methods

import (
	"fmt"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/compute"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/volume"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/volumeattach"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"regexp"
	"time"
)

type AttachDiskMethod struct {
	computeServiceBuilder compute.ComputeServiceBuilder
	volumeServiceBuilder  volume.VolumeServiceBuilder
	cpiConfig             config.CpiConfig
	logger                utils.Logger
}

func NewAttachDiskMethod(
	computeServiceBuilder compute.ComputeServiceBuilder,
	volumeServiceBuilder volume.VolumeServiceBuilder,
	cpiConfig config.CpiConfig,
	logger utils.Logger,
) AttachDiskMethod {
	return AttachDiskMethod{
		computeServiceBuilder: computeServiceBuilder,
		volumeServiceBuilder:  volumeServiceBuilder,
		cpiConfig:             cpiConfig,
		logger:                logger,
	}
}

const waitTimeSeconds = 60
const firstDeviceNameLetter = 'b'

var initialDiskHint = apiv1.DiskHint{}

func (a AttachDiskMethod) attachDisk(vmCID apiv1.VMCID, diskCID apiv1.DiskCID, returnDiskHint bool) (apiv1.DiskHint, error) {

	diskHint := initialDiskHint

	a.logger.Info("attach_disk", fmt.Sprintf("Execute attach disk ID %s to VM ID %s", diskCID.AsString(), vmCID.AsString()))
	volumeService, err := a.volumeServiceBuilder.Build()
	if err != nil {
		return diskHint, fmt.Errorf("attach_disk: Failed to get volume service (attach_disk): %w", err)
	}
	diskVolume, err := volumeService.GetVolume(diskCID.AsString())
	if err != nil {
		return diskHint, fmt.Errorf("attach_disk: Failed to get volume: %w", err)
	}
	if len(diskVolume.Attachments) == 1 && diskVolume.Attachments[0].ServerID == vmCID.AsString() {
		a.logger.Info("attach_disk", fmt.Sprintf("Volume ID %s is already attached to VM ID %s", diskCID.AsString(), vmCID.AsString()))
		if returnDiskHint {
			diskHint = a.getDiskHint(*diskVolume, nil)
		}
		return diskHint, nil
	}
	err = a.checkDiskAttach(*diskVolume, vmCID)
	if err != nil {
		return diskHint, fmt.Errorf("attach_disk: Disk cannot be attached: %w", err)
	}
	// attach disk to VM
	computeService, err := a.computeServiceBuilder.Build()
	if err != nil {
		return diskHint, fmt.Errorf("attach_disk: Failed to get compute service: %w", err)
	}
	server, err := computeService.GetServer(vmCID.AsString())
	if err != nil {
		return diskHint, fmt.Errorf("attach_disk: Failed to get VM %s: %w", vmCID.AsString(), err)
	}
	mountPoint, err := a.getMountPoint(computeService, *server)
	if err != nil {
		return diskHint, fmt.Errorf("attach_disk: Failed to get mount point: %w", err)
	}
	a.logger.Debug("attach_disk", fmt.Sprintf("Attaching volume ID: %s, server: %s, mountPoint: %s", diskCID.AsString(), vmCID.AsString(), mountPoint))
	volumeAttachment, err := computeService.AttachVolume(server.ID, diskCID.AsString(), mountPoint)
	if err != nil {
		return diskHint, fmt.Errorf("attach_disk: Failed to attach volume ID %s to VM ID %s: %w", diskVolume.ID, server.ID, err)
	}
	a.logger.Debug("attach_disk", fmt.Sprintf("Attaching volume DONE: Volume ID: %s, VM ID: %s, mountPoint: %s", volumeAttachment.VolumeID, volumeAttachment.ServerID, volumeAttachment.Device))
	a.logger.Debug("attach_disk", fmt.Sprintf("Waiting for volume ID %s to get in use by VM ID %s (time: %d secs)", diskCID.AsString(), vmCID.AsString(), waitTimeSeconds))
	err = volumeService.WaitForVolumeToBecomeStatus(diskCID.AsString(), time.Duration(waitTimeSeconds)*time.Second, "in-use")
	if err != nil {
		return diskHint, fmt.Errorf("attach_disk: Timeout on waiting to attach volume ID %s to VM %s (waiting: %d sec): %w", diskVolume.ID, server.ID, waitTimeSeconds, err)
	}
	a.logger.Info("attach_disk", fmt.Sprintf("Successfully attached volume ID %s to VM %s (Volume status now: 'in-use')", diskCID.AsString(), vmCID.AsString()))
	if returnDiskHint {
		diskHint = a.getDiskHint(*diskVolume, volumeService)
	}
	return diskHint, nil
}

func (a AttachDiskMethod) checkDiskAttach(diskVolume volumes.Volume, vmCID apiv1.VMCID) error {

	if (len(diskVolume.Attachments) > 1) || (len(diskVolume.Attachments) == 1 && diskVolume.Attachments[0].ServerID != vmCID.AsString()) {
		return fmt.Errorf("volume %s is attached to another VM", diskVolume.ID)
	}
	// volume statuses: https://docs.openstack.org/api-ref/block-storage/v3/index.html#volumes-volumes
	if diskVolume.Status != "available" {
		return fmt.Errorf("volume %s has not status 'available', current status is '%s'", diskVolume.ID, diskVolume.Status)
	}
	return nil
}

func (a AttachDiskMethod) getFirstDeviceNameLetter(computeService compute.ComputeService, server servers.Server) (rune, error) {
	inspectChar := firstDeviceNameLetter
	configDrive := a.cpiConfig.OpenStackConfig().ConfigDrive
	idStr, ok := server.Flavor["id"].(string)
	if !ok {
		a.logger.Warn("getFirstDeviceNameLetter", fmt.Sprintf("No flavor ID for server %s found", server.ID))
	}
	flavor, err := computeService.GetFlavorById(idStr)
	if err != nil {
		return ' ', fmt.Errorf("getFirstDeviceNameLetter: Failed to get flavor by ID %s: %w", idStr, err)
	}
	if flavor.Ephemeral > 0 {
		inspectChar = inspectChar + 1
		a.logger.Debug("getFirstDeviceNameLetter", fmt.Sprintf("Flavor ID %s has ephemeral disk. Switch device name letter: %c\n", flavor.ID, inspectChar))
	}
	if flavor.Swap > 0 {
		inspectChar = inspectChar + 1
		a.logger.Debug("getFirstDeviceNameLetter", fmt.Sprintf("Flavor ID %s has swap disk. Switch device name letter: %c\n", flavor.ID, inspectChar))
	}
	if configDrive == "disk" {
		inspectChar = inspectChar + 1
		a.logger.Debug("getFirstDeviceNameLetter", fmt.Sprintf("ConfigDrive is set to 'disk'. Switch device name letter: %c\n", inspectChar))
	}
	a.logger.Debug("getFirstDeviceNameLetter", fmt.Sprintf("Starting check for device letter: %c\n", inspectChar))
	return inspectChar, nil
}

func (a AttachDiskMethod) getMountPoint(computeService compute.ComputeService, server servers.Server) (string, error) {
	inspectChar, err := a.getFirstDeviceNameLetter(computeService, server)
	if err != nil {
		return "", fmt.Errorf("getMountPoint: Failed to get first device letter service: %w", err)
	}
	attachmentList, err := computeService.ListVolumeAttachments(server.ID)
	if err != nil {
		return "", fmt.Errorf("getMountPoint: Failed to get volume attachments for VM ID %s: %w", server.ID, err)
	}
	a.logger.Debug("getMountPoint", fmt.Sprintf("Attachments for VM ID %s", server.ID))
	for idx, attachment := range attachmentList {
		a.logger.Debug("getMountPoint", fmt.Sprintf("%d: Existing attachment: device: %s, volume ID: %s", idx+1, attachment.Device, attachment.VolumeID))
	}
	inspectChar, err = a.getDeviceChar(inspectChar, attachmentList)
	if err != nil {
		return "", fmt.Errorf("getMountPoint: failed to get device letter for server ID %s: %w", server.ID, err)
	}
	return fmt.Sprintf("/dev/sd%c", inspectChar), nil
}

func (a AttachDiskMethod) getDeviceChar(inspectChar rune, attachments []volumeattach.VolumeAttachment) (rune, error) {
	var deviceList []string
	for _, attachment := range attachments {
		deviceList = append(deviceList, attachment.Device)
	}
	for char := inspectChar; char <= 'z'; char++ {
		devicePattern := fmt.Sprintf("\\/dev\\/(?:sd|vd|xvd)%c", char)
		matcher := regexp.MustCompile(devicePattern)
		found := false
		for _, device := range deviceList {
			if matcher.MatchString(device) {
				found = true
				break
			}
		}
		if !found {
			return char, nil
		}
	}
	return ' ', fmt.Errorf("failed to get device letter")
}

func (a AttachDiskMethod) getDiskHint(diskVolume volumes.Volume, volumeService volume.VolumeService) apiv1.DiskHint {
	diskHint := initialDiskHint
	if volumeService != nil {
		currVolume, err := volumeService.GetVolume(diskVolume.ID)
		if err != nil {
			a.logger.Error("attach_disk", fmt.Sprintf("Failed to get volume: %v", err))
		} else {
			if currVolume.Attachments != nil && len(currVolume.Attachments) != 0 {
				attachment := currVolume.Attachments[0]
				diskHint = apiv1.NewDiskHintFromString(attachment.Device)
			}
		}
	} else {
		if diskVolume.Attachments != nil && len(diskVolume.Attachments) != 0 {
			attachment := diskVolume.Attachments[0]
			diskHint = apiv1.NewDiskHintFromString(attachment.Device)
		}
	}
	a.logger.Debug("attach_disk", fmt.Sprintf("Use disk hint value: %v", diskHint))
	return diskHint
}

func (a AttachDiskMethod) AttachDisk(vmCID apiv1.VMCID, diskCID apiv1.DiskCID) error {
	_, err := a.attachDisk(vmCID, diskCID, false)
	return err
}

func (a AttachDiskMethod) AttachDiskV2(vmCID apiv1.VMCID, diskCID apiv1.DiskCID) (apiv1.DiskHint, error) {
	diskHint, err := a.attachDisk(vmCID, diskCID, true)
	return diskHint, err
}
