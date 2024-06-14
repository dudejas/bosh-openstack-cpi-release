package properties_test

import (
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NetworkConfig", func() {

	Context("UpdateWithServerData", func() {

		Context("Update IP of default network", func() {
			It("Updates if the network is dynamic", func() {
				networkConfig := properties.NetworkConfig{
					DefaultNetwork: properties.Network{
						Type: "dynamic",
						IP:   "not-set",
					},
				}

				server := servers.Server{
					Addresses: map[string]interface{}{"public": []interface{}{"192.168.1.1", "2001:db8::1"}},
				}

				networkConfig.UpdateWithServerData(server)

				Expect(networkConfig.DefaultNetwork.IP).To(Equal("192.168.1.1"))
			})

			It("Skips the update if the network is NOT dynamic", func() {
				networkConfig := properties.NetworkConfig{
					DefaultNetwork: properties.Network{
						Type: "manual",
						IP:   "1.1.1.1",
					},
				}

				server := servers.Server{
					Addresses: map[string]interface{}{"public": []interface{}{"192.168.1.1", "2001:db8::1"}},
				}

				networkConfig.UpdateWithServerData(server)

				Expect(networkConfig.DefaultNetwork.IP).To(Equal("1.1.1.1"))
			})
		})
	})
})
