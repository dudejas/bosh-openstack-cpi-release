package vm_test

import (
	"encoding/binary"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/vm"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"net"
	"sort"
)

var _ = Describe("NetworkConfig", func() {
	var config vm.NetworkConfig

	BeforeEach(func() {
		config, _ = createNetworkConfig([]byte(`{
				"name1": {
					"type":    "manual",
					"ip":      "1.1.1.1",
					"default": ["gateway"],
					"cloud_properties": {"net_id": "the_net_id_1", "security_groups": ["security_group_1", "security_group_2"]}
				},
				"name2": {
					"type":    "manual",
					"ip":      "2.2.2.2",
					"cloud_properties": {"net_id": "the_net_id_2"}
				},
				"name3": {
					"type":    "vip",
					"ip":      "3.3.3.3",
					"cloud_properties": {"net_id": "the_net_id_3", "security_groups": ["security_group_3"]}
				},
				"name4": {
					"type":    "dynamic",
					"ip":      "4.4.4.4",
					"cloud_properties": {"net_id": "the_net_id_4"}
				}
			}`))
	})

	Context("NewNetworkConfig", func() {
		It("returns an error if a manual network is missing a netid", func() {
			_, err := createNetworkConfig([]byte(`{
				"name1": {
					"type":    "manual",
					"ip":      "",
					"cloud_properties": {"missing_net_id": ""}
				},
				"name2": {
					"type":    "manual",
					"ip":      "",
					"cloud_properties": {"net_id": "the_net_id_2"}
				}
			}`))

			Expect(err.Error()).To(Equal("invalid manual network configuration: manual network must have a net_id"))
		})

		It("returns an error if multiple vip networks exists", func() {
			_, err := createNetworkConfig([]byte(`{
				"name1": {
					"type":    "vip",
					"ip":      "",
					"cloud_properties": {}
				},
				"name2": {
					"type":    "vip",
					"ip":      "",
					"cloud_properties": {}
				}
			}`))

			Expect(err.Error()).To(Equal("invalid vip network configuration: only one vip should be defined per instance"))
		})

		It("returns an error if multiple dynamic networks exists", func() {
			_, err := createNetworkConfig([]byte(`{
				"name1": {
					"type":    "dynamic",
					"ip":      "",
					"cloud_properties": {}
				},
				"name2": {
					"type":    "dynamic",
					"ip":      "",
					"cloud_properties": {}
				}
			}`))

			Expect(err.Error()).To(Equal("invalid dynamic network configuration: only one dynamic should be defined per instance"))
		})

		It("returns an error if same net_id is used by multiple networks", func() {
			_, err := createNetworkConfig([]byte(`{
				"name1": {
					"type":    "manual",
					"ip":      "",
					"cloud_properties": {"net_id": "same_net_id"}
				},
				"name2": {
					"type":    "dynamic",
					"ip":      "",
					"cloud_properties": {"net_id": "same_net_id"}
				}
			}`))

			Expect(err.Error()).To(Equal("invalid network configuration: network with id same_net_id is defined multiple times"))
		})
	})

	Context("GetDefaultNetwork", func() {
		It("returns the default network", func() {
			defaultNetwork := config.GetDefaultNetwork()

			Expect(defaultNetwork.Type).To(Equal("manual"))
			Expect(defaultNetwork.IP).To(Equal("1.1.1.1"))
			Expect(defaultNetwork.CloudProps.NetID).To(Equal("the_net_id_1"))
		})

		It("returns an empty network if no network is provided", func() {
			config, err := createNetworkConfig([]byte(`{}`))
			Expect(err).ToNot(HaveOccurred())
			defaultNetwork := config.GetDefaultNetwork()

			Expect(defaultNetwork.Type).To(Equal(""))
			Expect(defaultNetwork.IP).To(Equal(""))
			Expect(defaultNetwork.CloudProps.NetID).To(Equal(""))
		})
	})

	Context("GetManualNetworks", func() {
		It("returns the manual networks", func() {
			manualNetworks := sortNetworks(config.GetManualNetworks())

			Expect(manualNetworks[0].Type).To(Equal("manual"))
			Expect(manualNetworks[0].IP).To(Equal("1.1.1.1"))
			Expect(manualNetworks[0].CloudProps.NetID).To(Equal("the_net_id_1"))

			Expect(manualNetworks[1].Type).To(Equal("manual"))
			Expect(manualNetworks[1].IP).To(Equal("2.2.2.2"))
			Expect(manualNetworks[1].CloudProps.NetID).To(Equal("the_net_id_2"))
		})
	})

	Context("GetVIPNetwork", func() {
		It("returns the vip network", func() {
			vipNetwork := config.GetVIPNetwork()

			Expect(vipNetwork.Type).To(Equal("vip"))
			Expect(vipNetwork.IP).To(Equal("3.3.3.3"))
			Expect(vipNetwork.CloudProps.NetID).To(Equal("the_net_id_3"))
		})
	})

	Context("GetDynamicNetwork", func() {
		It("returns the dynamic network", func() {
			dynamicNetwork := config.GetDynamicNetwork()

			Expect(dynamicNetwork.Type).To(Equal("dynamic"))
			Expect(dynamicNetwork.IP).To(Equal("4.4.4.4"))
			Expect(dynamicNetwork.CloudProps.NetID).To(Equal("the_net_id_4"))
		})
	})

	Context("GetAllNetworks", func() {
		It("returns all networks", func() {
			allNetworks := sortNetworks(config.GetAllNetworks())

			Expect(allNetworks[0].Type).To(Equal("manual"))
			Expect(allNetworks[0].IP).To(Equal("1.1.1.1"))
			Expect(allNetworks[0].CloudProps.NetID).To(Equal("the_net_id_1"))

			Expect(allNetworks[1].Type).To(Equal("manual"))
			Expect(allNetworks[1].IP).To(Equal("2.2.2.2"))
			Expect(allNetworks[1].CloudProps.NetID).To(Equal("the_net_id_2"))

			Expect(allNetworks[2].Type).To(Equal("vip"))
			Expect(allNetworks[2].IP).To(Equal("3.3.3.3"))
			Expect(allNetworks[2].CloudProps.NetID).To(Equal("the_net_id_3"))

			Expect(allNetworks[3].IP).To(Equal("4.4.4.4"))
			Expect(allNetworks[3].CloudProps.NetID).To(Equal("the_net_id_4"))
			Expect(allNetworks[3].Type).To(Equal("dynamic"))
		})
	})

	Context("SecurityGroups", func() {
		It("returns all security groups", func() {
			securityGroups := config.SecurityGroups()

			Expect(securityGroups[0]).To(Equal("security_group_1"))
			Expect(securityGroups[1]).To(Equal("security_group_2"))
			Expect(securityGroups[2]).To(Equal("security_group_3"))
		})
	})
})

func sortNetworks(networks []vm.Network) []vm.Network {
	sort.Slice(networks, func(i, j int) bool {
		return ip2int(net.ParseIP(networks[i].IP)) < ip2int(net.ParseIP(networks[j].IP))
	})

	return networks
}

func ip2int(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}
