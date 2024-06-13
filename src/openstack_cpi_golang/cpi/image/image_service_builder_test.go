package image_test

import (
	"errors"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/image"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/openstack/openstackfakes"
	"github.com/gophercloud/gophercloud"

	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils/utilsfakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ImageServiceBuilder", func() {
	var openstackService openstackfakes.FakeOpenstackService
	var logger utilsfakes.FakeLogger
	var imageServiceBuilder image.ImageServiceBuilder

	BeforeEach(func() {
		openstackService = openstackfakes.FakeOpenstackService{}
		logger = utilsfakes.FakeLogger{}
		imageServiceBuilder = image.NewImageServiceBuilder(
			&openstackService,
			config.OpenstackConfig{},
			&logger,
		)
	})

	Context("Build", func() {
		It("returns an image service", func() {
			openstackService.ImageServiceV2Returns(&gophercloud.ServiceClient{}, nil)

			computeService, err := imageServiceBuilder.Build()

			Expect(err).ToNot(HaveOccurred())
			Expect(computeService).To(Not(BeNil()))
		})

		It("returns an error if the compute service client cannot be retrieved", func() {
			openstackService.ImageServiceV2Returns(nil, errors.New("boom"))

			computeService, err := imageServiceBuilder.Build()

			Expect(err.Error()).To(Equal("failed to retrieve image service client: boom"))
			Expect(computeService).To(BeNil())
		})
	})
})
