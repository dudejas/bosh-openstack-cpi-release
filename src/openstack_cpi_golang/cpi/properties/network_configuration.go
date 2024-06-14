package properties

import "github.com/gophercloud/gophercloud/openstack/compute/v2/servers"

type NetworkConfig struct {
	DefaultNetwork Network
	ManualNetworks []Network
	VIPNetwork     *Network
	DynamicNetwork *Network
	SecurityGroups []string
}

type Network struct {
	Type       string
	CloudProps CreateVMNetwork
	IP         string
}

func (n *NetworkConfig) UpdateWithServerData(server servers.Server) {
	n.updateDefaultNetwork(server)
}

func (n *NetworkConfig) updateDefaultNetwork(server servers.Server) {

	if n.DefaultNetwork.Type != "dynamic" {
		return
	}

	for _, addressList := range server.Addresses {
		if addresses, ok := addressList.([]interface{}); ok {
			for _, address := range addresses {
				if ip, ok := address.(string); ok {
					n.DefaultNetwork.IP = ip
					return
				}
			}
		}
	}
}
