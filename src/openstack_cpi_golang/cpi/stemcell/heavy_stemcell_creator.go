package stemcell

import (
	"fmt"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/cloud_properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services"
)

//counterfeiter:generate . HeavyStemcellCreator
type HeavyStemcellCreator interface {
	Create(imageService services.ImageService, cloudProps cloud_properties.CreateStemcell, imagePath string) (string, error)
}

type heavyStemcellCreator struct {
	config config.OpenstackConfig
}

func NewHeavyStemcellCreator(
	config config.OpenstackConfig,
) heavyStemcellCreator {
	return heavyStemcellCreator{
		config: config,
	}
}

func (h heavyStemcellCreator) Create(imageService services.ImageService, cloudProps cloud_properties.CreateStemcell, rootImagePath string) (string, error) {
	imageID, err := imageService.CreateImage(cloudProps, h.config)
	if err != nil {
		return "", fmt.Errorf("failed to create image: %w", err)
	}

	err = imageService.UploadImage(imageID, rootImagePath)
	if err != nil {
		return "", fmt.Errorf("failed to upload root image: %w", err)
	}

	return imageID, nil
}
