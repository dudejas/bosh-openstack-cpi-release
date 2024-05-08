package services

import (
	"fmt"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/clients"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services/facades"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . ImageServiceBuilder
type ImageServiceBuilder interface {
	Build() (ImageService, error)
}

type imageServiceBuilder struct {
	openstackService OpenstackService
	openstackConfig  config.OpenstackConfig
}

func NewImageServiceBuilder(openstackService OpenstackService, openstackConfig config.OpenstackConfig) imageServiceBuilder {
	return imageServiceBuilder{
		openstackService: openstackService,
		openstackConfig:  openstackConfig,
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
	), nil
}
