package services_test

import (
	"errors"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services/facades/facadesfakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils/utilsfakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/vm"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type MockPage struct{}

func (m MockPage) NextPageURL() (string, error) {
	return "", nil
}
func (m MockPage) IsEmpty() (bool, error) {
	return false, nil
}
func (m MockPage) GetBody() interface{} {
	return nil
}

var _ = Describe("NetworkService", func() {
	var networkConfig vm.NetworkConfig
	var serviceClient gophercloud.ServiceClient
	var networkingFacade facadesfakes.FakeNetworkingFacade
	var logger utilsfakes.FakeLogger
	var floatingIpPage MockPage
	var portPage MockPage

	BeforeEach(func() {
		providerClient := gophercloud.ProviderClient{TokenID: "the_token"}
		serviceClient = gophercloud.ServiceClient{ProviderClient: &providerClient}
		networkingFacade = facadesfakes.FakeNetworkingFacade{}
		logger = utilsfakes.FakeLogger{}
		floatingIpPage = MockPage{}
		portPage = MockPage{}

		networkingFacade.ListFloatingIpsReturns(floatingIpPage, nil)
		networkingFacade.ExtractFloatingIPsReturns([]floatingips.FloatingIP{{ID: "the_floating_ip_id"}}, nil)
		networkingFacade.ListPortsReturns(portPage, nil)
		networkingFacade.ExtractPortsReturns([]ports.Port{{ID: "5678"}}, nil)

		networkConfig, _ = createNetworkConfig([]byte(`{
				"name1": {
					"type":    "manual",
					"ip":      "1.1.1.1",
					"default": ["gateway"],
					"cloud_properties": {"net_id": "the_net_id_1", "security_groups": ["security_group_1", "security_group_2"]}
				},
				"name3": {
					"type":    "vip",
					"ip":      "3.3.3.3",
					"cloud_properties": {"net_id": "the_net_id_3", "security_groups": ["security_group_3"]}
				}
			}`))
	})

	Context("ConfigureNetwork", func() {
		It("lists floating ips", func() {
			services.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureNetwork("123-456", networkConfig)

			_, listOpts := networkingFacade.ListFloatingIpsArgsForCall(0)
			Expect(listOpts.FloatingIP).To(Equal("3.3.3.3"))
		})

		It("returns an error if floating ips cannot be fetched from openstack", func() {
			networkingFacade.ListFloatingIpsReturns(nil, errors.New("boom"))

			err := services.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureNetwork("123-456", networkConfig)
			Expect(err.Error()).To(Equal("failed to get floating IP: failed to list floating IPs: boom"))
		})

		It("extracts floating ips", func() {
			services.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureNetwork("123-456", networkConfig)

			pages := networkingFacade.ExtractFloatingIPsArgsForCall(0)
			Expect(pages).To(Equal(floatingIpPage))
		})

		It("returns an error if floating ips cannot be extracted from pages", func() {
			networkingFacade.ExtractFloatingIPsReturns(nil, errors.New("boom"))

			err := services.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureNetwork("123-456", networkConfig)
			Expect(err.Error()).To(Equal("failed to get floating IP: failed to extract floating IPs: boom"))
		})

		It("returns an error if floating ips are empty", func() {
			networkingFacade.ExtractFloatingIPsReturns([]floatingips.FloatingIP{}, nil)

			err := services.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureNetwork("123-456", networkConfig)
			Expect(err.Error()).To(Equal("failed to get floating IP: floating IP 3.3.3.3 not allocated"))
		})

		It("lists ports", func() {
			services.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureNetwork("123-456", networkConfig)

			_, listOpts := networkingFacade.ListPortsArgsForCall(0)
			Expect(listOpts.DeviceID).To(Equal("123-456"))
			Expect(listOpts.NetworkID).To(Equal("the_net_id_1"))
		})

		It("returns an error if port listing fails", func() {
			networkingFacade.ListPortsReturns(nil, errors.New("boom"))

			err := services.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureNetwork("123-456", networkConfig)
			Expect(err.Error()).To(Equal("failed to get port: failed to list ports: boom"))
		})

		It("extracts ports", func() {
			services.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureNetwork("123-456", networkConfig)

			pages := networkingFacade.ExtractPortsArgsForCall(0)
			Expect(pages).To(Equal(portPage))
		})

		It("returns an error if ports cannot be extracted from pages", func() {
			networkingFacade.ExtractPortsReturns(nil, errors.New("boom"))

			err := services.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureNetwork("123-456", networkConfig)
			Expect(err.Error()).To(Equal("failed to get port: failed to extract ports: boom"))
		})

		It("returns an error if ports are empty", func() {
			networkingFacade.ExtractPortsReturns([]ports.Port{}, nil)

			err := services.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureNetwork("123-456", networkConfig)
			Expect(err.Error()).To(Equal("failed to get port: no port allocated by instance 123-456 and network the_net_id_1"))
		})

		It("associates the floating ip to a port", func() {
			services.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureNetwork("123-456", networkConfig)

			_, floatingIpId, updateOpts := networkingFacade.UpdateFloatingIPArgsForCall(0)
			Expect(floatingIpId).To(Equal("the_floating_ip_id"))
			Expect(*updateOpts.PortID).To(Equal("5678"))
		})

		It("returns an error if port association fails", func() {
			networkingFacade.UpdateFloatingIPReturns(nil, errors.New("boom"))

			err := services.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureNetwork("123-456", networkConfig)
			Expect(err.Error()).To(Equal("failed to associate floating ip to port: boom"))
		})
	})

})
