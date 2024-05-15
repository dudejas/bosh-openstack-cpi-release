package methods

import (
	"errors"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/cloud_properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services/servicesfakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/stemcell/root_image/root_imagefakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/stemcell/stemcellfakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils/utilsfakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type MockStemcellCloudProps struct {
	ImageID string
	Version string
}

func (m *MockStemcellCloudProps) As(v interface{}) error {
	stemcell := v.(*cloud_properties.CreateStemcell)
	stemcell.Version = "0.1"
	stemcell.ImageID = m.ImageID

	return nil
}

var _ = Describe("CreateStemcellMethod", func() {

	var serviceFactory servicesfakes.FakeServiceFactory
	var heavyStemcellCreator stemcellfakes.FakeHeavyStemcellCreator
	var lightStemcellCreator stemcellfakes.FakeLightStemcellCreator
	var rootImageProvider root_imagefakes.FakeRootImage
	var logger utilsfakes.FakeLogger

	Context("CreateStemcell", func() {

		BeforeEach(func() {
			serviceFactory = servicesfakes.FakeServiceFactory{}
			heavyStemcellCreator = stemcellfakes.FakeHeavyStemcellCreator{}
			lightStemcellCreator = stemcellfakes.FakeLightStemcellCreator{}
			rootImageProvider = root_imagefakes.FakeRootImage{}
			logger = utilsfakes.FakeLogger{}
		})

		It("returns a stemcell ID", func() {
			serviceFactory.CreateImageServiceReturns(services.NewImageService(nil, nil, nil, nil), nil)
			heavyStemcellCreator.CreateReturns("123-456", nil)
			props := &MockStemcellCloudProps{}
			stemcellCID, err := NewCreateStemcellMethod(
				&serviceFactory,
				&heavyStemcellCreator,
				&lightStemcellCreator,
				&rootImageProvider,
				config.OpenstackConfig{},
				&logger,
			).CreateStemcell("imagePath", props)

			Expect(err).ToNot(HaveOccurred())
			Expect(stemcellCID.AsString()).To(Equal("123-456"))
		})

		It("returns an error if the stemcell creation fails", func() {
			serviceFactory.CreateImageServiceReturns(services.NewImageService(nil, nil, nil, nil), nil)
			heavyStemcellCreator.CreateReturns("", errors.New("boom"))
			props := &MockStemcellCloudProps{}
			stemcellCID, err := NewCreateStemcellMethod(
				&serviceFactory,
				&heavyStemcellCreator,
				&lightStemcellCreator,
				&rootImageProvider,
				config.OpenstackConfig{},
				&logger,
			).CreateStemcell("imagePath", props)

			Expect(err.Error()).To(Equal("failed to create a stemcell: boom"))
			Expect(stemcellCID).To(Equal(apiv1.StemcellCID{}))
		})

		It("returns an error if the image service cannot be retrieved", func() {
			serviceFactory.CreateImageServiceReturns(nil, errors.New("boom"))
			props := &MockStemcellCloudProps{}
			stemcellCID, err := NewCreateStemcellMethod(
				&serviceFactory,
				&heavyStemcellCreator,
				&lightStemcellCreator,
				&rootImageProvider,
				config.OpenstackConfig{},
				&logger,
			).CreateStemcell("imagePath", props)

			Expect(err.Error()).To(Equal("failed to create image service: boom"))
			Expect(stemcellCID).To(Equal(apiv1.StemcellCID{}))
		})

		It("uses the light stemcell creation if cloud properties are containing an imageID", func() {
			theImageService := services.NewImageService(nil, nil, nil, nil)
			serviceFactory.CreateImageServiceReturns(theImageService, nil)
			lightStemcellCreator.CreateReturns("123-456", nil)

			theCloudProps := &MockStemcellCloudProps{ImageID: "123-456"}

			NewCreateStemcellMethod(
				&serviceFactory,
				&heavyStemcellCreator,
				&lightStemcellCreator,
				&rootImageProvider,
				config.OpenstackConfig{},
				&logger,
			).CreateStemcell("imagePath", theCloudProps)

			imageService, cloudProps := lightStemcellCreator.CreateArgsForCall(0)
			Expect(lightStemcellCreator.CreateCallCount()).To(Equal(1))
			Expect(imageService).To(Equal(theImageService))
			Expect(cloudProps.ImageID).To(Equal("123-456"))
		})

		It("uses the heavy stemcell creation if cloud properties are NOT containing an imageID", func() {
			theImageService := services.NewImageService(nil, nil, nil, nil)
			serviceFactory.CreateImageServiceReturns(theImageService, nil)
			heavyStemcellCreator.CreateReturns("123-456", nil)
			rootImageProvider.GetReturns("rootImagePath", nil)

			theCloudProps := &MockStemcellCloudProps{}

			NewCreateStemcellMethod(
				&serviceFactory,
				&heavyStemcellCreator,
				&lightStemcellCreator,
				&rootImageProvider,
				config.OpenstackConfig{},
				&logger,
			).CreateStemcell("imagePath", theCloudProps)

			imageService, cloudProps, path := heavyStemcellCreator.CreateArgsForCall(0)
			Expect(heavyStemcellCreator.CreateCallCount()).To(Equal(1))
			Expect(imageService).To(Equal(theImageService))
			Expect(cloudProps.Version).To(Equal("0.1"))
			Expect(path).To(Equal("rootImagePath"))
		})

		It("returns an error if root.img cannot be retrieved", func() {
			theImageService := services.NewImageService(nil, nil, nil, nil)
			serviceFactory.CreateImageServiceReturns(theImageService, nil)
			rootImageProvider.GetReturns("", errors.New("boom"))

			theCloudProps := &MockStemcellCloudProps{}

			rootImagePath, err := NewCreateStemcellMethod(
				&serviceFactory,
				&heavyStemcellCreator,
				&lightStemcellCreator,
				&rootImageProvider,
				config.OpenstackConfig{},
				&logger,
			).CreateStemcell("imagePath", theCloudProps)

			Expect(err.Error()).To(Equal("failed to get root image: boom"))
			Expect(rootImagePath).To(Equal(apiv1.StemcellCID{}))
		})

		It("extracts the rootImage to a temp dir path", func() {
			theImageService := services.NewImageService(nil, nil, nil, nil)
			serviceFactory.CreateImageServiceReturns(theImageService, nil)
			rootImageProvider.GetReturns("", errors.New("boom"))

			theCloudProps := &MockStemcellCloudProps{}

			NewCreateStemcellMethod(
				&serviceFactory,
				&heavyStemcellCreator,
				&lightStemcellCreator,
				&rootImageProvider,
				config.OpenstackConfig{},
				&logger,
			).CreateStemcell("imagePath", theCloudProps)

			imagePath, tempDirPath := rootImageProvider.GetArgsForCall(0)
			Expect(imagePath).To(Equal("imagePath"))
			Expect(tempDirPath).To(MatchRegexp("/tmp/unpacked-image-\\d+"))
		})

	})
})
