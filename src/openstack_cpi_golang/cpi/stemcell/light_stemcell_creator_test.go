package stemcell

import (
	"fmt"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/cloud_properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services/servicesfakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("lightStemcellCreator", func() {
	Context("Create", func() {
		It("returns the stemcell id of an existing image", func() {
			imageServiceClient := servicesfakes.FakeImageService{}
			imageServiceClient.GetImageReturns("1234", nil)
			subject := NewLightStemcellCreator(config.OpenstackConfig{})
			imageID, err := subject.Create(&imageServiceClient, cloud_properties.CreateStemcell{})

			Expect(err).ToNot(HaveOccurred())
			Expect(imageID).To(Equal("1234"))
		})

		It("returns an error if no image can be found", func() {
			imageServiceClient := servicesfakes.FakeImageService{}
			imageServiceClient.GetImageReturns("", fmt.Errorf("boom"))
			subject := NewLightStemcellCreator(config.OpenstackConfig{})
			imageID, err := subject.Create(&imageServiceClient, cloud_properties.CreateStemcell{})

			Expect(err.Error()).To(Equal("failed to retrieve image: boom"))
			Expect(imageID).To(Equal(""))
		})

		It("gets an image via imageID", func() {
			imageServiceClient := servicesfakes.FakeImageService{}

			subject := NewLightStemcellCreator(config.OpenstackConfig{})
			subject.Create(&imageServiceClient, cloud_properties.CreateStemcell{ImageID: "123-456"})

			imageID := imageServiceClient.GetImageArgsForCall(0)
			Expect(imageID).To(Equal("123-456"))
		})
	})
})
