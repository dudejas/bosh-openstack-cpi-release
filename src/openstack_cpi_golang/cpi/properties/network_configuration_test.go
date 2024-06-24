package properties_test

import (
	"encoding/json"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NetworkConfig", func() {

	Context("AllNetworks", func() {

		var manualNetwork1 properties.Network
		var manualNetwork2 properties.Network
		var vipNetwork *properties.Network
		var dynamicNetwork *properties.Network

		BeforeEach(func() {
			manualNetwork1 = properties.Network{Type: "manual", Key: "1"}
			manualNetwork2 = properties.Network{Type: "manual", Key: "2"}
			vipNetwork = &properties.Network{Type: "vip", Key: "3"}
			dynamicNetwork = &properties.Network{Type: "dynamic", Key: "4"}
		})

		It("returns a list of manual, dynamic and vip networks", func() {
			networkConfig := properties.NetworkConfig{
				ManualNetworks: []properties.Network{manualNetwork1, manualNetwork2},
				VIPNetwork:     vipNetwork,
				DynamicNetwork: dynamicNetwork,
			}

			Expect(networkConfig.AllNetworks()).To(ContainElements(manualNetwork1, manualNetwork2, *vipNetwork, *dynamicNetwork))
		})

		It("skips adding dynamic network, if not defined", func() {
			networkConfig := properties.NetworkConfig{
				ManualNetworks: []properties.Network{manualNetwork1, manualNetwork2},
				VIPNetwork:     vipNetwork,
			}

			Expect(networkConfig.AllNetworks()).To(ContainElements(manualNetwork1, manualNetwork2, *vipNetwork))
		})

		It("skips adding vip network, if not defined", func() {
			networkConfig := properties.NetworkConfig{
				ManualNetworks: []properties.Network{manualNetwork1, manualNetwork2},
				DynamicNetwork: dynamicNetwork,
			}

			Expect(networkConfig.AllNetworks()).To(ContainElements(manualNetwork1, manualNetwork2, *dynamicNetwork))
		})
	})

	Context("UpdateWithServerData", func() {
		var server servers.Server

		jsonStr := `{
					"addresses": {
						"public": [
							{"version": 4, "addr": "192.168.1.1"},
							{"version": 6, "addr": "2001:db8::1"}
						]
					}
				}`

		Context("Update IP of default network", func() {
			It("updates if the network is dynamic", func() {
				networkConfig := properties.NetworkConfig{
					DefaultNetwork: properties.Network{
						Type: "dynamic",
						IP:   "not-set",
					},
				}

				err := json.Unmarshal([]byte(jsonStr), &server)
				Expect(err).ToNot(HaveOccurred())

				networkConfig.UpdateWithServerData(server)

				Expect(networkConfig.DefaultNetwork.IP).To(Equal("192.168.1.1"))
			})

			It("skips the update if the network is NOT dynamic", func() {
				networkConfig := properties.NetworkConfig{
					DefaultNetwork: properties.Network{
						Type: "manual",
						IP:   "1.1.1.1",
					},
				}

				networkConfig.UpdateWithServerData(server)

				Expect(networkConfig.DefaultNetwork.IP).To(Equal("1.1.1.1"))
			})
		})
	})

	Context("AsNetworkSpec", func() {

		var manualNetwork1 properties.Network
		var vipNetwork *properties.Network
		var dynamicNetwork *properties.Network

		var networkConfig properties.NetworkConfig

		BeforeEach(func() {
			manualNetwork1 = properties.Network{
				Key:        "key-1",
				Default:    []string{"gateway"},
				DNS:        []string{"8.8.8.8"},
				IP:         "1.2.3.4",
				Gateway:    "1.2.3.1",
				Netmask:    "255.255.255.0",
				Type:       "manual",
				CloudProps: properties.NetworkCloudProps{NetID: "net-id-1", SecurityGroups: []string{"sg-1", "sg-2"}},
			}
			vipNetwork = &properties.Network{
				Key:  "key-2",
				Type: "vip",
			}
			dynamicNetwork = &properties.Network{
				Key:        "key-3",
				Type:       "dynamic",
				CloudProps: properties.NetworkCloudProps{NetID: "net-id-3"},
			}
		})

		It("returns a spec with a single network", func() {
			networkConfig = properties.NetworkConfig{
				ManualNetworks: []properties.Network{manualNetwork1},
			}

			spec, err := networkConfig.AsNetworkSpec()

			Expect(err).ToNot(HaveOccurred())
			Expect(spec).To(HaveLen(1))
			Expect(spec["key-1"]).NotTo(BeNil())
			Expect(spec["key-1"].Default()).To(Equal([]string{"gateway"}))
			Expect(spec["key-1"].DNS()).To(Equal([]string{"8.8.8.8"}))
			Expect(spec["key-1"].IP()).To(Equal("1.2.3.4"))
			Expect(spec["key-1"].Gateway()).To(Equal("1.2.3.1"))
			Expect(spec["key-1"].Netmask()).To(Equal("255.255.255.0"))
			Expect(spec["key-1"].Type()).To(Equal("manual"))

			cloudProps := properties.NetworkCloudProps{}
			spec["key-1"].CloudProps().As(&cloudProps)

			Expect(cloudProps.NetID).To(Equal("net-id-1"))
			Expect(cloudProps.SecurityGroups).To(Equal([]string{"sg-1", "sg-2"}))
		})

		It("returns a spec with multiple networks", func() {
			networkConfig = properties.NetworkConfig{
				ManualNetworks: []properties.Network{manualNetwork1},
				VIPNetwork:     vipNetwork,
				DynamicNetwork: dynamicNetwork,
			}

			spec, err := networkConfig.AsNetworkSpec()

			Expect(err).ToNot(HaveOccurred())
			Expect(spec).To(HaveLen(3))
			Expect(spec["key-1"].Type()).To(Equal("manual"))
			Expect(spec["key-2"].Type()).To(Equal("vip"))
			Expect(spec["key-3"].Type()).To(Equal("dynamic"))

			cloudProps1 := properties.NetworkCloudProps{}
			spec["key-1"].CloudProps().As(&cloudProps1)
			Expect(cloudProps1.NetID).To(Equal("net-id-1"))
			Expect(cloudProps1.SecurityGroups).To(Equal([]string{"sg-1", "sg-2"}))

			cloudProps2 := properties.NetworkCloudProps{}
			spec["key-2"].CloudProps().As(&cloudProps2)
			Expect(cloudProps2.NetID).To(Equal(""))
			Expect(cloudProps2.SecurityGroups).To(BeNil())

			cloudProps3 := properties.NetworkCloudProps{}
			spec["key-3"].CloudProps().As(&cloudProps3)
			Expect(cloudProps3.NetID).To(Equal("net-id-3"))
			Expect(cloudProps3.SecurityGroups).To(BeNil())
		})
	})
})
