package methods

import (
	"errors"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services/servicesfakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils/utilsfakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeleteStemcellMethod", func() {

	var serviceFactory servicesfakes.FakeServiceFactory
	var logger utilsfakes.FakeLogger
	var imageService servicesfakes.FakeImageService

	Context("DeleteStemcell", func() {

		BeforeEach(func() {
			serviceFactory = servicesfakes.FakeServiceFactory{}
			logger = utilsfakes.FakeLogger{}
		})

		It("deletes an image ID", func() {
			serviceFactory.CreateImageServiceReturns(&imageService, nil)
			imageService.DeleteImageReturns(nil)
			err := NewDeleteStemcellMethod(
				&serviceFactory,
				&logger,
			).DeleteStemcell(apiv1.NewStemcellCID("cloudID"))

			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if image cannot be deleted", func() {
			serviceFactory.CreateImageServiceReturns(&imageService, nil)
			imageService.DeleteImageReturns(errors.New("boom"))
			err := NewDeleteStemcellMethod(
				&serviceFactory,
				&logger,
			).DeleteStemcell(apiv1.NewStemcellCID("cloudID"))

			Expect(err.Error()).To(Equal("failed to delete stemcell with cid cloudID due to the following: boom"))
		})

		It("returns an error if the image service cannot be retrieved", func() {
			serviceFactory.CreateImageServiceReturns(nil, errors.New("boom"))
			err := NewDeleteStemcellMethod(
				&serviceFactory,
				&logger,
			).DeleteStemcell(apiv1.NewStemcellCID("cloudID"))

			Expect(err.Error()).To(Equal("failed to create image service: boom"))
		})
	})
})
