package methods

import (
	"fmt"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/vm"
)

type CreateVMMethod struct {
	serviceFactory services.ServiceFactory
	config         config.OpenstackConfig
	logger         utils.Logger
}

func NewCreateVMMethod(serviceFactory services.ServiceFactory, config config.OpenstackConfig, logger utils.Logger) CreateVMMethod {
	return CreateVMMethod{
		serviceFactory: serviceFactory,
		config:         config,
		logger:         logger,
	}
}

func (m CreateVMMethod) CreateVM(
	agentID apiv1.AgentID, stemcellCID apiv1.StemcellCID, cloudProps apiv1.VMCloudProps,
	networks apiv1.Networks, diskCIDs []apiv1.DiskCID, env apiv1.VMEnv) (apiv1.VMCID, error) {

	return apiv1.VMCID{}, nil
}

func (m CreateVMMethod) CreateVMV2(
	agentID apiv1.AgentID, stemcellCID apiv1.StemcellCID, props apiv1.VMCloudProps,
	networks apiv1.Networks, diskCIDs []apiv1.DiskCID, env apiv1.VMEnv) (apiv1.VMCID, apiv1.Networks, error) {

	cloudProps := properties.CreateVM{}
	props.As(&cloudProps)

	computeService, err := m.serviceFactory.CreateComputeService()
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, fmt.Errorf("failed to create compute service: %w", err)
	}

	networkService, err := m.serviceFactory.CreateNetworkService()
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, fmt.Errorf("failed to create networking service: %w", err)
	}

	imageService, err := m.serviceFactory.CreateImageService()
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, fmt.Errorf("failed to create image service: %w", err)
	}

	_, err = imageService.GetImage(stemcellCID.AsString())
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, fmt.Errorf("failed to resolve stemcell: %w", err)
	}

	networkConfig, err := vm.NewNetworkConfig(networks)
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, fmt.Errorf("failed to create network config: %w", err)
	}

	serverID, err := computeService.CreateServer(stemcellCID, cloudProps, networkConfig, m.config)
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, fmt.Errorf("failed to create server: %w", err)
	}

	err = networkService.ConfigureNetwork(serverID, networkConfig)
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, fmt.Errorf("failed to configure network for server %s: %w", serverID, err)
	}

	return apiv1.NewVMCID(serverID), networks, nil
}
