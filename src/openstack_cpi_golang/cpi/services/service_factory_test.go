package services_test

import (
	"errors"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services"
	"github.com/gophercloud/gophercloud"

	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services/servicesfakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils/utilsfakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ServiceFactory", func() {
	var openstackService servicesfakes.FakeOpenstackService
	var logger utilsfakes.FakeLogger
	var serviceFactory services.ServiceFactory

	BeforeEach(func() {
		openstackService = servicesfakes.FakeOpenstackService{}
		logger = utilsfakes.FakeLogger{}
		serviceFactory = services.NewServiceFactory(
			&openstackService,
			config.OpenstackConfig{},
			&logger,
		)
	})

	Context("CreateComputeService", func() {
		It("returns a compute service", func() {
			openstackService.ComputeServiceV2Returns(&gophercloud.ServiceClient{}, nil)

			computeService, err := serviceFactory.CreateComputeService()

			Expect(err).ToNot(HaveOccurred())
			Expect(computeService).To(Not(BeNil()))
		})

		It("returns an error if the compute service client cannot be retrieved", func() {
			openstackService.ComputeServiceV2Returns(nil, errors.New("boom"))

			computeService, err := serviceFactory.CreateComputeService()

			Expect(err.Error()).To(Equal("failed to retrieve compute service client: boom"))
			Expect(computeService).To(BeNil())
		})
	})

	Context("CreateNetworkService", func() {
		It("returns a network service", func() {
			openstackService.NetworkServiceV2Returns(&gophercloud.ServiceClient{}, nil)

			computeService, err := serviceFactory.CreateNetworkService()

			Expect(err).ToNot(HaveOccurred())
			Expect(computeService).To(Not(BeNil()))
		})

		It("returns an error if the compute service client cannot be retrieved", func() {
			openstackService.NetworkServiceV2Returns(nil, errors.New("boom"))

			computeService, err := serviceFactory.CreateNetworkService()

			Expect(err.Error()).To(Equal("failed to retrieve network service client: boom"))
			Expect(computeService).To(BeNil())
		})
	})

	Context("CreateImageService", func() {
		It("returns an image service", func() {
			openstackService.ImageServiceV2Returns(&gophercloud.ServiceClient{}, nil)

			computeService, err := serviceFactory.CreateImageService()

			Expect(err).ToNot(HaveOccurred())
			Expect(computeService).To(Not(BeNil()))
		})

		It("returns an error if the compute service client cannot be retrieved", func() {
			openstackService.ImageServiceV2Returns(nil, errors.New("boom"))

			computeService, err := serviceFactory.CreateImageService()

			Expect(err.Error()).To(Equal("failed to retrieve image service client: boom"))
			Expect(computeService).To(BeNil())
		})
	})
})
