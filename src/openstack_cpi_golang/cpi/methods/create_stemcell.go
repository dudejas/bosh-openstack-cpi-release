package methods

import (
	"fmt"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/stemcell"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/stemcell/root_image"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	"os"
)

type CreateStemcellMethod struct {
	serviceFactory       services.ServiceFactory
	heavyStemcellCreator stemcell.HeavyStemcellCreator
	lightStemcellCreator stemcell.LightStemcellCreator
	rootImageProvider    root_image.RootImage
	config               config.OpenstackConfig
	logger               utils.Logger
}

func NewCreateStemcellMethod(
	serviceFactory services.ServiceFactory,
	heavyStemcellCreator stemcell.HeavyStemcellCreator,
	lightStemcellCreator stemcell.LightStemcellCreator,
	rootImageProvider root_image.RootImage,
	config config.OpenstackConfig,
	logger utils.Logger,
) CreateStemcellMethod {
	return CreateStemcellMethod{
		serviceFactory:       serviceFactory,
		heavyStemcellCreator: heavyStemcellCreator,
		lightStemcellCreator: lightStemcellCreator,
		rootImageProvider:    rootImageProvider,
		config:               config,
		logger:               logger,
	}
}

func (a CreateStemcellMethod) CreateStemcell(
	imagePath string,
	props apiv1.StemcellCloudProps,
) (apiv1.StemcellCID, error) {
	a.logger.Info("create_stemcell", "Creating new image...")

	var cloudProps = properties.CreateStemcell{}
	props.As(&cloudProps)

	imageService, err := a.serviceFactory.CreateImageService()
	if err != nil {
		return apiv1.StemcellCID{}, fmt.Errorf("failed to create image service: %w", err)
	}

	var imageID string
	var creationError error
	if cloudProps.ImageID != "" {
		imageID, creationError = a.lightStemcellCreator.Create(imageService, cloudProps)

	} else {
		tempDirPath, err := os.MkdirTemp("/tmp", "unpacked-image-")
		if err != nil {
			return apiv1.StemcellCID{}, fmt.Errorf("failed to create temp dir: %w", err)
		}
		defer os.RemoveAll(tempDirPath)

		rootImagePath, err := a.rootImageProvider.Get(imagePath, tempDirPath)
		if err != nil {
			return apiv1.StemcellCID{}, fmt.Errorf("failed to get root image: %w", err)
		}

		imageID, creationError = a.heavyStemcellCreator.Create(imageService, cloudProps, rootImagePath)
	}

	if creationError != nil {
		return apiv1.StemcellCID{}, fmt.Errorf("failed to create a stemcell: %w", creationError)
	}
	return apiv1.NewStemcellCID(imageID), nil
}
