package methods

import (
	"fmt"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/compute"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/network"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
)

type DeleteVMMethod struct {
	networkServiceBuilder network.NetworkServiceBuilder
	computeServiceBuilder compute.ComputeServiceBuilder
	config                config.OpenstackConfig
	logger                utils.Logger
}

func NewDeleteVMMethod(
	networkServiceBuilder network.NetworkServiceBuilder,
	computeServiceBuilder compute.ComputeServiceBuilder,
	config config.OpenstackConfig,
	logger utils.Logger,
) DeleteVMMethod {
	return DeleteVMMethod{
		networkServiceBuilder: networkServiceBuilder,
		computeServiceBuilder: computeServiceBuilder,
		config:                config,
		logger:                logger,
	}
}

func (a DeleteVMMethod) DeleteVM(cid apiv1.VMCID) error {
	computeService, err := a.computeServiceBuilder.Build()
	if err != nil {
		return fmt.Errorf("failed to create compute service: %w", err)
	}

	networkService, err := a.networkServiceBuilder.Build()
	if err != nil {
		return fmt.Errorf("failed to create network service: %w", err)
	}

	// Get ports before deleting the server so that it is still assigned to the server
	ports, err := networkService.GetPorts(cid.AsString(), properties.Network{}, true)
	if err != nil {
		return fmt.Errorf("failed to get ports: %w", err)
	}

	err = computeService.DeleteServer(cid.AsString(), a.config)
	if err != nil {
		return fmt.Errorf("failed to delete server: %w", err)
	}

	err = networkService.DeletePorts(ports)
	if err != nil {
		return fmt.Errorf("failed to delete ports: %w", err)
	}

	return nil
}
