package network

import (
	"fmt"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/openstack"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
)

//counterfeiter:generate . NetworkServiceBuilder
type NetworkServiceBuilder interface {
	Build() (NetworkService, error)
}

type networkServiceBuilder struct {
	openstackService openstack.OpenstackService
	openstackConfig  config.OpenstackConfig
	logger           utils.Logger
}

func NewNetworkServiceBuilder(openstackService openstack.OpenstackService, openstackConfig config.OpenstackConfig, logger utils.Logger) networkServiceBuilder {
	return networkServiceBuilder{
		openstackService: openstackService,
		openstackConfig:  openstackConfig,
		logger:           logger,
	}
}

func (b networkServiceBuilder) Build() (NetworkService, error) {
	serviceClient, err := b.openstackService.NetworkServiceV2(b.openstackConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve network service client: %w", err)
	}

	return NewNetworkService(
		serviceClient,
		NewNetworkingFacade(),
		b.logger,
	), nil
}
