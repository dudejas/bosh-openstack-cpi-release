package properties

import (
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
)

type NetworkConfig struct {
	DefaultNetwork Network
	ManualNetworks []Network
	VIPNetwork     *Network
	DynamicNetwork *Network
	SecurityGroups []string
}

type NetworksMap map[string]Network

type Network struct {
	Key        string
	Default    []string          `json:"default"`
	DNS        []string          `json:"dns"`
	IP         string            `json:"ip,omitempty"`
	Gateway    string            `json:"gateway,omitempty"`
	Netmask    string            `json:"netmask,omitempty"`
	Type       string            `json:"type"`
	CloudProps NetworkCloudProps `json:"cloud_properties"`
}

type NetworkCloudProps struct {
	NetID          string   `json:"net_id,omitempty"`
	SecurityGroups []string `json:"security_groups,omitempty"`
}

func (n *NetworkConfig) AllNetworks() []Network {
	networks := n.ManualNetworks

	if n.DynamicNetwork != nil {
		networks = append(networks, *n.DynamicNetwork)
	}

	if n.VIPNetwork != nil {
		networks = append(networks, *n.VIPNetwork)
	}

	return networks
}

func (n *NetworkConfig) UpdateWithServerData(server servers.Server) {
	n.updateDefaultNetwork(server)
}

func (n *NetworkConfig) AsNetworkSpec() (apiv1.Networks, error) {
	networks := apiv1.Networks{}

	networksMap := NetworksMap{}
	for _, network := range n.AllNetworks() {
		networksMap[network.Key] = network
	}
	networksJson, err := json.Marshal(networksMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal networks: %w", err)
	}

	//As side effect this fills the networks
	err = networks.UnmarshalJSON(networksJson)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal networks: %w", err)
	}

	return networks, nil
}

func (n *NetworkConfig) updateDefaultNetwork(server servers.Server) {

	if n.DefaultNetwork.Type != "dynamic" {
		return
	}

	for _, addressList := range server.Addresses {
		if addresses, ok := addressList.([]interface{}); ok {
			for _, address := range addresses {
				if addr, ok := address.(map[string]interface{}); ok {
					n.DefaultNetwork.IP = addr["addr"].(string)
					return
				}
			}
		}
	}
}
