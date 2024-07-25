package volume_test

import (
	"errors"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/volume"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/volume/volumefakes"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"time"
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
		volumeFacade.CreateDiskReturns(&volumes.Volume{ID: "123-456"}, nil)
		defaultCloudConfig = properties.CreateDisk{VolumeType: "the_volume_type"}
	})

	Context("CreateVolume", func() {

		It("returns error if volume was failed to be created", func() {
			volumeFacade.CreateDiskReturns(&volumes.Volume{ID: "123-456", Status: "available"}, errors.New("boom"))

			_, err := volumeService.CreateVolume(1, defaultCloudConfig, "z1")

			Expect(err.Error()).To(Equal("failed to create volume: boom"))
		})

		It("returns an active server", func() {
			volumeFacade.CreateDiskReturns(&volumes.Volume{ID: "123-456", Status: "available"}, nil)

			volume, err := volumeService.CreateVolume(1, defaultCloudConfig, "z1")

			Expect(err).ToNot(HaveOccurred())
			Expect(volume).ToNot(BeNil())
		})
	})

	Context("WaitForVolumeToBecomeAvailable", func() {
		It("returns error if volume was failed to become available", func() {
			volumeFacade.GetVolumeReturns(&volumes.Volume{ID: "123-456", Status: "error"}, nil)

			_, err := volumeService.WaitForVolumeToBecomeAvailable("123-456", 1*time.Second)

			Expect(err.Error()).To(Equal("volume became error state while waiting to become available"))
		})

		It("returns an available volume", func() {
			volumeFacade.GetVolumeReturnsOnCall(0, &volumes.Volume{ID: "123-456", Status: "creating"}, nil)
			volumeFacade.GetVolumeReturnsOnCall(1, &volumes.Volume{ID: "123-456", Status: "available"}, nil)

			volume, err := volumeService.WaitForVolumeToBecomeAvailable("123-456", 1*time.Second)

			Expect(volumeFacade.GetVolumeCallCount()).To(Equal(2))
			Expect(err).ToNot(HaveOccurred())
			Expect(volume).ToNot(BeNil())
		})

		It("times out while waiting for volume to become available", func() {
			volumeFacade.GetVolumeReturns(&volumes.Volume{ID: "123-456", Status: "creating"}, nil)

			_, err := volumeService.WaitForVolumeToBecomeAvailable("123-456", 0)

			Expect(err.Error()).To(Equal("timeout while waiting for volume to become available"))
		})

		It("returns an error if it cannot get the volume", func() {
			volumeFacade.GetVolumeReturns(&volumes.Volume{}, errors.New("boom"))

			_, err := volumeService.WaitForVolumeToBecomeAvailable("123-456", 0)

			Expect(volumeFacade.GetVolumeCallCount()).To(Equal(1))
			Expect(err.Error()).To(Equal("boom"))
		})
	})
})
