package compute

import (
	"fmt"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/openstack"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
)

//counterfeiter:generate . ComputeServiceBuilder
type ComputeServiceBuilder interface {
	Build() (ComputeService, error)
}

type computeServiceBuilder struct {
	openstackService openstack.OpenstackService
	openstackConfig  config.OpenstackConfig
	logger           utils.Logger
}

func NewComputeServiceBuilder(openstackService openstack.OpenstackService, openstackConfig config.OpenstackConfig, logger utils.Logger) computeServiceBuilder {
	return computeServiceBuilder{
		openstackService: openstackService,
		openstackConfig:  openstackConfig,
		logger:           logger,
	}
}

func (b computeServiceBuilder) Build() (ComputeService, error) {
	serviceClient, err := b.openstackService.ComputeServiceV2(b.openstackConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve compute service client: %w", err)
	}

	return NewComputeService(
		serviceClient,
		NewComputeFacade(),
		b.logger,
	), nil
}
