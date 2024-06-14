package methods

import (
	"fmt"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/compute"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/image"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/network"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
)

type CreateVMMethod struct {
	imageServiceBuilder   image.ImageServiceBuilder
	networkServiceBuilder network.NetworkServiceBuilder
	computeServiceBuilder compute.ComputeServiceBuilder
	config                config.OpenstackConfig
	logger                utils.Logger
}

func NewCreateVMMethod(
	imageServiceBuilder image.ImageServiceBuilder,
	networkServiceBuilder network.NetworkServiceBuilder,
	computeServiceBuilder compute.ComputeServiceBuilder,
	config config.OpenstackConfig,
	logger utils.Logger,
) CreateVMMethod {
	return CreateVMMethod{
		imageServiceBuilder:   imageServiceBuilder,
		networkServiceBuilder: networkServiceBuilder,
		computeServiceBuilder: computeServiceBuilder,
		config:                config,
		logger:                logger,
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

	computeService, err := m.computeServiceBuilder.Build()
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, fmt.Errorf("failed to create compute service: %w", err)
	}

	networkService, err := m.networkServiceBuilder.Build()
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, fmt.Errorf("failed to create networking service: %w", err)
	}

	imageService, err := m.imageServiceBuilder.Build()
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, fmt.Errorf("failed to create image service: %w", err)
	}

	_, err = imageService.GetImage(stemcellCID.AsString())
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, fmt.Errorf("failed to resolve stemcell: %w", err)
	}

	networkConfig, err := networkService.GetNetworkConfiguration(networks, m.config, cloudProps)
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, fmt.Errorf("failed to create network config: %w", err)
	}

	server, err := computeService.CreateServer(stemcellCID, cloudProps, networkConfig, m.config)
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, fmt.Errorf("failed to create server: %w", err)
	}

	networkConfig.UpdateWithServerData(*server)

	err = networkService.ConfigureVIPNetwork(server.ID, networkConfig)
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, fmt.Errorf("failed to configure network for server %s: %w", server.ID, err)
	}

	return apiv1.NewVMCID(server.ID), networks, nil
}
