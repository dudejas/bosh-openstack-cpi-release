package services

import (
	"fmt"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/clients"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services/facades"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
)

//counterfeiter:generate . ServiceFactory
type ServiceFactory interface {
	CreateImageService() (ImageService, error)
}

type serviceFactory struct {
	openstackService OpenstackService
	openstackConfig  config.OpenstackConfig
	logger           utils.Logger
}

func NewServiceFactory(openstackService OpenstackService, openstackConfig config.OpenstackConfig, logger utils.Logger) serviceFactory {
	return serviceFactory{
		openstackService: openstackService,
		openstackConfig:  openstackConfig,
		logger:           logger,
	}
}

func (b serviceFactory) CreateImageService() (ImageService, error) {
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
