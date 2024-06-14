package properties_test

import (
	"encoding/json"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NetworkConfig", func() {

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
})
