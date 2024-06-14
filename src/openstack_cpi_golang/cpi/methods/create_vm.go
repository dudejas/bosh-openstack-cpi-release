package methods

import (
	"fmt"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/compute"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/image"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/loadbalancer"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/network"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/pools"
	"strconv"
)

type CreateVMMethod struct {
	imageServiceBuilder        image.ImageServiceBuilder
	networkServiceBuilder      network.NetworkServiceBuilder
	computeServiceBuilder      compute.ComputeServiceBuilder
	loadbalancerServiceBuilder loadbalancer.LoadbalancerServiceBuilder
	config                     config.OpenstackConfig
	logger                     utils.Logger
}

func NewCreateVMMethod(
	imageServiceBuilder image.ImageServiceBuilder,
	networkServiceBuilder network.NetworkServiceBuilder,
	computeServiceBuilder compute.ComputeServiceBuilder,
	loadbalancerServiceBuilder loadbalancer.LoadbalancerServiceBuilder,
	config config.OpenstackConfig,
	logger utils.Logger,
) CreateVMMethod {
	return CreateVMMethod{
		imageServiceBuilder:        imageServiceBuilder,
		networkServiceBuilder:      networkServiceBuilder,
		computeServiceBuilder:      computeServiceBuilder,
		loadbalancerServiceBuilder: loadbalancerServiceBuilder,
		config:                     config,
		logger:                     logger,
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

	err := cloudProps.Validate()
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, fmt.Errorf("failed to validate cloud properties: %w", err)
	}

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

	loadbalancerService, err := m.loadbalancerServiceBuilder.Build()
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, fmt.Errorf("failed to create loadbalancer service: %w", err)
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
		return apiv1.VMCID{}, apiv1.Networks{}, fmt.Errorf("failed to configure network for server '%s': %w", server.ID, err)
	}

	poolMembers, err := m.configureLoadbalancerPools(loadbalancerService, networkService, cloudProps, networkConfig)
	if err != nil {
		return apiv1.VMCID{}, apiv1.Networks{}, fmt.Errorf("failed to configure loadbalancer pools: %w", err)
	}

	computeService.SetMetadata(*server, m.getServerTags(poolMembers))

	return apiv1.NewVMCID(server.ID), networks, nil
}

func (m CreateVMMethod) configureLoadbalancerPools(
	loadbalancerService loadbalancer.LoadbalancerService,
	networkService network.NetworkService,
	cloudProps properties.CreateVM,
	networkConfig properties.NetworkConfig,
) ([]pools.Member, error) {
	poolMemberships := []pools.Member{}
	for _, pool := range cloudProps.LoadbalancerPools {

		poolID, err := loadbalancerService.GetPoolID(pool.Name)
		if err != nil {
			return []pools.Member{}, fmt.Errorf("failed to get pool ID of pool '%s': %w", pool.Name, err)
		}

		ip := networkConfig.DefaultNetwork.IP

		defaultNetworkID := networkConfig.DefaultNetwork.CloudProps.NetID

		subnetID, err := networkService.GetSubnetID(defaultNetworkID, ip)
		if err != nil {
			return []pools.Member{}, fmt.Errorf("failed to get subnet: %w", err)
		}

		poolMember, err := loadbalancerService.CreatePoolMember(poolID, ip, pool, subnetID, m.config.StateTimeOut)
		if err != nil {
			return []pools.Member{}, fmt.Errorf("failed to create pool membership of IP '%s' in pool '%s': %w", ip, pool.Name, err)
		}

		poolMemberships = append(poolMemberships, *poolMember)
	}
	return poolMemberships, nil
}

func (m CreateVMMethod) getServerTags(members []pools.Member) properties.ServerTags {
	tags := properties.ServerTags{}

	var index = 1
	for _, member := range members {
		itoa := strconv.Itoa(index)
		tags["lbaas_pool_"+itoa] = member.PoolID + "/" + member.ID
		index++
	}

	return tags
}
