package vm

import (
	"fmt"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"slices"
)

type networkConfig struct {
	DefaultNetwork Network
	ManualNetworks []Network
	VIPNetwork     *Network
	DynamicNetwork *Network
}

//counterfeiter:generate . NetworkConfig
type NetworkConfig interface {
	GetAllNetworks() []Network

	SecurityGroups() []string

	GetDefaultNetwork() Network

	GetManualNetworks() []Network

	GetVIPNetwork() *Network

	GetDynamicNetwork() *Network
}

type Network struct {
	Type       string
	CloudProps properties.CreateVMNetwork
	IP         string
}

func NewNetworkConfig(networks apiv1.Networks) (NetworkConfig, error) {

	defaultNetwork := createNetwork(networks.Default())

	manualNetworks, err := createManuelNetwork(networks)
	if err != nil {
		return networkConfig{}, fmt.Errorf("invalid manual network configuration: %w", err)
	}

	vipNetwork, err := createSingleNetwork(networks, "vip")
	if err != nil {
		return networkConfig{}, fmt.Errorf("invalid vip network configuration: %w", err)
	}

	dynamicNetwork, err := createSingleNetwork(networks, "dynamic")
	if err != nil {
		return networkConfig{}, fmt.Errorf("invalid dynamic network configuration: %w", err)
	}

	resultNetworkConfig := networkConfig{
		DefaultNetwork: defaultNetwork,
		ManualNetworks: manualNetworks,
		VIPNetwork:     vipNetwork,
		DynamicNetwork: dynamicNetwork,
	}

	err = validateNetIDs(resultNetworkConfig.manualAndDynamicNetworks())
	if err != nil {
		return networkConfig{}, fmt.Errorf("invalid network configuration: %w", err)
	}

	return resultNetworkConfig, nil
}

func (nc networkConfig) GetDefaultNetwork() Network {
	return nc.DefaultNetwork
}

func (nc networkConfig) GetManualNetworks() []Network {
	return nc.ManualNetworks
}

func (nc networkConfig) GetVIPNetwork() *Network {
	return nc.VIPNetwork
}

func (nc networkConfig) GetDynamicNetwork() *Network {
	return nc.DynamicNetwork
}

func (nc networkConfig) SecurityGroups() []string {
	var securityGroups []string
	var uniqueSecurityGroups []string
	securityGroupsMap := make(map[string]bool)

	for _, network := range nc.GetAllNetworks() {
		securityGroups = append(securityGroups, network.CloudProps.SecurityGroups...)
	}

	for _, entry := range securityGroups {
		if _, value := securityGroupsMap[entry]; !value {
			securityGroupsMap[entry] = true
			uniqueSecurityGroups = append(uniqueSecurityGroups, entry)
		}
	}

	return uniqueSecurityGroups
}

func (nc networkConfig) GetAllNetworks() []Network {
	var networks []Network

	networks = append(networks, nc.manualAndDynamicNetworks()...)

	if nc.VIPNetwork != nil {
		networks = append(networks, *nc.VIPNetwork)
	}

	return networks
}

func (nc networkConfig) manualAndDynamicNetworks() []Network {
	var networks []Network

	networks = append(networks, nc.ManualNetworks...)

	if nc.DynamicNetwork != nil {
		networks = append(networks, *nc.DynamicNetwork)
	}

	return networks
}

func createManuelNetwork(networks apiv1.Networks) ([]Network, error) {
	var manualNetworks []Network

	for _, network := range networks {
		if network.Type() == "manual" {
			createdNetwork := createNetwork(network)

			netID := createdNetwork.CloudProps.NetID
			if netID == "" {
				return []Network{}, fmt.Errorf("manual network must have a net_id")
			}

			manualNetworks = append(manualNetworks, createdNetwork)
		}
	}

	return manualNetworks, nil
}

func createSingleNetwork(networks apiv1.Networks, networkType string) (*Network, error) {
	var network *Network

	for _, net := range networks {
		if net.Type() == networkType {
			if network != nil {
				return &Network{}, fmt.Errorf("only one %s should be defined per instance", networkType)
			}

			createdNetwork := createNetwork(net)
			network = &createdNetwork
		}
	}

	return network, nil
}

func createNetwork(network apiv1.Network) Network {
	vmNetworkProps := properties.CreateVMNetwork{}
	network.CloudProps().As(&vmNetworkProps)

	return Network{
		Type:       network.Type(),
		CloudProps: vmNetworkProps,
		IP:         network.IP(),
	}
}

func validateNetIDs(networks []Network) error {
	var usedNetIDs []string

	for _, network := range networks {
		netID := network.CloudProps.NetID
		if slices.Contains(usedNetIDs, netID) {
			return fmt.Errorf("network with id %s is defined multiple times", netID)
		}

		usedNetIDs = append(usedNetIDs, netID)
	}

	return nil
}
