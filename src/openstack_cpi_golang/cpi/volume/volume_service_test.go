package volume_test

import (
	"errors"
	"time"

	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/volume"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/volume/volumefakes"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("VolumeService", func() {
	var serviceClient gophercloud.ServiceClient
	var retryableServiceClient gophercloud.ServiceClient
	var serviceClients utils.ServiceClients
	var volumeFacade volumefakes.FakeVolumeFacade
	var defaultCloudConfig properties.CreateDisk
	var volumeService volume.VolumeService

	BeforeEach(func() {
		serviceClient = gophercloud.ServiceClient{}
		retryableServiceClient = gophercloud.ServiceClient{}
		serviceClients = utils.ServiceClients{ServiceClient: &serviceClient, RetryableServiceClient: &retryableServiceClient}
		volumeFacade = volumefakes.FakeVolumeFacade{}

		volumeService = volume.NewVolumeService(serviceClients, &volumeFacade)
		volume.VolumeServicePollingInterval = 0
		volumeFacade.CreateVolumeReturns(&volumes.Volume{ID: "123-456"}, nil)
		defaultCloudConfig = properties.CreateDisk{VolumeType: "the_volume_type"}
	})

	Context("CreateVolume", func() {

		It("returns error if volume was failed to be created", func() {
			volumeFacade.CreateVolumeReturns(&volumes.Volume{ID: "123-456", Status: "available"}, errors.New("boom"))

			_, err := volumeService.CreateVolume(1, defaultCloudConfig, "z1")

			Expect(err.Error()).To(Equal("failed to create volume: boom"))
		})

		It("returns an available volume", func() {
			volumeFacade.CreateVolumeReturns(&volumes.Volume{ID: "123-456", Status: "available"}, nil)

			volume, err := volumeService.CreateVolume(1, defaultCloudConfig, "z1")

			Expect(err).ToNot(HaveOccurred())
			Expect(volume).ToNot(BeNil())
		})
	})

	Context("WaitForVolumeToBecomeStatus", func() {
		It("returns error if volume was failed to become available", func() {
			volumeFacade.GetVolumeReturns(&volumes.Volume{ID: "123-456", Status: "error"}, nil)

			err := volumeService.WaitForVolumeToBecomeStatus("123-456", 1*time.Second, "some_target_status")

			Expect(err.Error()).To(Equal("volume became error state while waiting to become some_target_status"))
		})

		It("returns an available volume", func() {
			volumeFacade.GetVolumeReturnsOnCall(0, &volumes.Volume{ID: "123-456", Status: "creating"}, nil)
			volumeFacade.GetVolumeReturnsOnCall(1, &volumes.Volume{ID: "123-456", Status: "some_target_status"}, nil)

			err := volumeService.WaitForVolumeToBecomeStatus("123-456", 1*time.Second, "some_target_status")

			Expect(volumeFacade.GetVolumeCallCount()).To(Equal(2))
			Expect(err).ToNot(HaveOccurred())
		})

		It("times out while waiting for volume to become some_target_status", func() {
			volumeFacade.GetVolumeReturns(&volumes.Volume{ID: "123-456", Status: "creating"}, nil)

			err := volumeService.WaitForVolumeToBecomeStatus("123-456", 1, "some_target_status")

			Expect(err.Error()).To(Equal("timeout while waiting for volume to become some_target_status"))
		})

		It("returns an error if it cannot get the volume", func() {
			volumeFacade.GetVolumeReturns(&volumes.Volume{}, errors.New("boom"))

			err := volumeService.WaitForVolumeToBecomeStatus("123-456", 1, "some_target_status")

			Expect(volumeFacade.GetVolumeCallCount()).To(Equal(1))
			Expect(err.Error()).To(Equal("failed to retrieve volume information: boom"))
		})
	})

	Context("DeleteVolume", func() {

		It("returns error if volume was failed to be deleted", func() {
			volumeFacade.DeleteVolumeReturns(errors.New("boom"))

			err := volumeService.DeleteVolume("some_disk_cid")

			Expect(err.Error()).To(Equal("failed to delete volume: boom"))
		})

		It("returns nil if deletion of volume was successful", func() {
			volumeFacade.DeleteVolumeReturns(nil)

			err := volumeService.DeleteVolume("some_disk_cid")

			Expect(err).ToNot(HaveOccurred())
		})
	})
})
