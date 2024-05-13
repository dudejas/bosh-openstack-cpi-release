package services

import (
	"fmt"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/clients"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services/facades"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . ImageServiceBuilder
type ImageServiceBuilder interface {
	Build() (ImageService, error)
}

type imageServiceBuilder struct {
	openstackService OpenstackService
	openstackConfig  config.OpenstackConfig
	logger           utils.Logger
}

func NewImageServiceBuilder(openstackService OpenstackService, openstackConfig config.OpenstackConfig, logger utils.Logger) imageServiceBuilder {
	return imageServiceBuilder{
		openstackService: openstackService,
		openstackConfig:  openstackConfig,
		logger:           logger,
	}
}

func (b imageServiceBuilder) Build() (ImageService, error) {
	serviceClient, err := b.openstackService.ImageServiceV2(b.openstackConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve image service client: %w", err)
	}

	return NewImageService(
		serviceClient,
		facades.NewImagesFacade(),
		clients.NewHttpClient(),
		b.logger,
	), nil
}
