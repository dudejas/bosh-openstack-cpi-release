package network_test

import (
	"errors"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/mocks"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/network"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/network/networkfakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils/utilsfakes"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NetworkService", func() {
	var networkConfig properties.NetworkConfig
	var serviceClient gophercloud.ServiceClient
	var networkingFacade networkfakes.FakeNetworkingFacade
	var logger utilsfakes.FakeLogger
	var floatingIpPage mocks.MockPage
	var portPage mocks.MockPage

	BeforeEach(func() {
		providerClient := gophercloud.ProviderClient{TokenID: "the_token"}
		serviceClient = gophercloud.ServiceClient{ProviderClient: &providerClient}
		networkingFacade = networkfakes.FakeNetworkingFacade{}
		logger = utilsfakes.FakeLogger{}
		floatingIpPage = mocks.MockPage{}
		portPage = mocks.MockPage{}

		networkingFacade.ListFloatingIpsReturns(floatingIpPage, nil)
		networkingFacade.ExtractFloatingIPsReturns([]floatingips.FloatingIP{{ID: "the_floating_ip_id"}}, nil)
		networkingFacade.ListPortsReturns(portPage, nil)
		networkingFacade.ExtractPortsReturns([]ports.Port{{ID: "5678"}}, nil)

		networkConfig = properties.NetworkConfig{
			DefaultNetwork: properties.Network{CloudProps: properties.CreateVMNetwork{NetID: "the_net_id_1"}},
			VIPNetwork:     &properties.Network{IP: "3.3.3.3"},
		}
	})

	Context("ConfigureVIPNetwork", func() {
		It("lists floating ips", func() {
			network.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureVIPNetwork("123-456", networkConfig)

			_, listOpts := networkingFacade.ListFloatingIpsArgsForCall(0)
			Expect(listOpts.FloatingIP).To(Equal("3.3.3.3"))
		})

		It("returns an error if floating ips cannot be fetched from openstack", func() {
			networkingFacade.ListFloatingIpsReturns(nil, errors.New("boom"))

			err := network.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureVIPNetwork("123-456", networkConfig)
			Expect(err.Error()).To(Equal("failed to get floating IP: failed to list floating IPs: boom"))
		})

		It("extracts floating ips", func() {
			network.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureVIPNetwork("123-456", networkConfig)

			pages := networkingFacade.ExtractFloatingIPsArgsForCall(0)
			Expect(pages).To(Equal(floatingIpPage))
		})

		It("returns an error if floating ips cannot be extracted from pages", func() {
			networkingFacade.ExtractFloatingIPsReturns(nil, errors.New("boom"))

			err := network.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureVIPNetwork("123-456", networkConfig)
			Expect(err.Error()).To(Equal("failed to get floating IP: failed to extract floating IPs: boom"))
		})

		It("returns an error if floating ips are empty", func() {
			networkingFacade.ExtractFloatingIPsReturns([]floatingips.FloatingIP{}, nil)

			err := network.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureVIPNetwork("123-456", networkConfig)
			Expect(err.Error()).To(Equal("failed to get floating IP: floating IP 3.3.3.3 not allocated"))
		})

		It("lists ports", func() {
			network.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureVIPNetwork("123-456", networkConfig)

			_, listOpts := networkingFacade.ListPortsArgsForCall(0)
			Expect(listOpts.DeviceID).To(Equal("123-456"))
			Expect(listOpts.NetworkID).To(Equal("the_net_id_1"))
		})

		It("returns an error if port listing fails", func() {
			networkingFacade.ListPortsReturns(nil, errors.New("boom"))

			err := network.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureVIPNetwork("123-456", networkConfig)
			Expect(err.Error()).To(Equal("failed to get port: failed to list ports: boom"))
		})

		It("extracts ports", func() {
			network.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureVIPNetwork("123-456", networkConfig)

			pages := networkingFacade.ExtractPortsArgsForCall(0)
			Expect(pages).To(Equal(portPage))
		})

		It("returns an error if ports cannot be extracted from pages", func() {
			networkingFacade.ExtractPortsReturns(nil, errors.New("boom"))

			err := network.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureVIPNetwork("123-456", networkConfig)
			Expect(err.Error()).To(Equal("failed to get port: failed to extract ports: boom"))
		})

		It("returns an error if ports are empty", func() {
			networkingFacade.ExtractPortsReturns([]ports.Port{}, nil)

			err := network.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureVIPNetwork("123-456", networkConfig)
			Expect(err.Error()).To(Equal("failed to get port: no port allocated by instance 123-456 and network the_net_id_1"))
		})

		It("associates the floating ip to a port", func() {
			network.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureVIPNetwork("123-456", networkConfig)

			_, floatingIpId, updateOpts := networkingFacade.UpdateFloatingIPArgsForCall(0)
			Expect(floatingIpId).To(Equal("the_floating_ip_id"))
			Expect(*updateOpts.PortID).To(Equal("5678"))
		})

		It("returns an error if port association fails", func() {
			networkingFacade.UpdateFloatingIPReturns(nil, errors.New("boom"))

			err := network.NewNetworkService(&serviceClient, &networkingFacade, &logger).ConfigureVIPNetwork("123-456", networkConfig)
			Expect(err.Error()).To(Equal("failed to associate floating ip to port: boom"))
		})
	})
})
