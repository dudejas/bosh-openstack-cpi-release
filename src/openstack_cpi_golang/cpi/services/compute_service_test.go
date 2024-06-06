package services_test

import (
	"errors"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services/facades/facadesfakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services/servicesmocks"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils/utilsfakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/vm"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/vm/vmfakes"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ComputeService", func() {
	var serviceClient gophercloud.ServiceClient
	var computeFacade facadesfakes.FakeComputeFacade
	var logger utilsfakes.FakeLogger
	var computeService services.ComputeService
	var networkConfig vmfakes.FakeNetworkConfig
	var flavorsPage servicesmocks.MockPage

	BeforeEach(func() {
		providerClient := gophercloud.ProviderClient{TokenID: "the_token"}
		serviceClient = gophercloud.ServiceClient{ProviderClient: &providerClient}
		computeFacade = facadesfakes.FakeComputeFacade{}
		logger = utilsfakes.FakeLogger{}
		computeService = services.NewComputeService(&serviceClient, &computeFacade, &logger)
		networkConfig = vmfakes.FakeNetworkConfig{}
		flavorsPage = servicesmocks.MockPage{}

		computeFacade.CreateServerReturns(&servers.Server{ID: "123-456"}, nil)
		computeFacade.GetServerReturns(&servers.Server{ID: "123-456", Status: "ACTIVE"}, nil)
		computeFacade.ListFlavorsReturns(flavorsPage, nil)
		computeFacade.ExtractFlavorsReturns([]flavors.Flavor{{ID: "the_flavor_id", Name: "the_instance_type", RAM: 4096, Ephemeral: 10}}, nil)
	})

	Context("CreateServer", func() {

		It("list flavors", func() {
			_, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				properties.CreateVM{InstanceType: "the_instance_type"},
				&networkConfig,
				config.OpenstackConfig{StateTimeOut: 10},
			)
			Expect(err).ToNot(HaveOccurred())

			Expect(computeFacade.ListFlavorsCallCount()).To(Equal(1))
		})

		It("return error if list flavors fails", func() {
			computeFacade.ListFlavorsReturns(nil, errors.New("boom"))

			_, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				properties.CreateVM{InstanceType: "the_instance_type"},
				&networkConfig,
				config.OpenstackConfig{StateTimeOut: 10},
			)

			Expect(err.Error()).To(ContainSubstring("failed to list flavors: boom"))
		})

		It("extract flavors", func() {
			computeService.CreateServer(
				apiv1.StemcellCID{},
				properties.CreateVM{InstanceType: "the_instance_type"},
				&networkConfig,
				config.OpenstackConfig{StateTimeOut: 10},
			)

			Expect(computeFacade.ExtractFlavorsCallCount()).To(Equal(1))
		})

		It("return error if extract flavors fails", func() {
			computeFacade.ExtractFlavorsReturns(nil, errors.New("boom"))

			_, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				properties.CreateVM{InstanceType: "the_instance_type"},
				&networkConfig,
				config.OpenstackConfig{StateTimeOut: 10},
			)

			Expect(err.Error()).To(ContainSubstring("failed to extract flavors: boom"))
		})

		It("return an error if flavor name is not found", func() {
			_, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				properties.CreateVM{InstanceType: "not_existing_flavor"},
				&networkConfig,
				config.OpenstackConfig{StateTimeOut: 10},
			)

			Expect(err.Error()).To(ContainSubstring("flavor 'not_existing_flavor' not found"))
		})

		It("return an error if flavor ephemeral disk is to small", func() {
			computeFacade.ExtractFlavorsReturns([]flavors.Flavor{{ID: "the_flavor_id", Name: "the_instance_type", RAM: 4096, Ephemeral: 2}}, nil)

			_, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				properties.CreateVM{InstanceType: "the_flavor_id"},
				&networkConfig,
				config.OpenstackConfig{StateTimeOut: 10},
			)

			Expect(err.Error()).To(ContainSubstring("failed to get flavor of instance type: flavor 'the_flavor_id' not found"))
		})

		It("creates ops for the server", func() {
			networkConfig.GetManualNetworksReturns(
				[]vm.Network{
					{IP: "1.2.3.4", CloudProps: properties.CreateVMNetwork{NetID: "the_net_id"}},
				},
			)
			networkConfig.SecurityGroupsReturns([]string{"group_1", "group_2"})

			_, err := computeService.CreateServer(
				apiv1.NewStemcellCID("the_stemcell_id"),
				properties.CreateVM{AvailabilityZone: "the_availability_zone"},
				&networkConfig,
				config.OpenstackConfig{DefaultKeyName: "the_key_name"},
			)
			if err != nil {
				return
			}
			Expect(err).ToNot(HaveOccurred())

			sClient, opts := computeFacade.CreateServerArgsForCall(0)
			Expect(sClient).To(Equal(&serviceClient))
			createMap, err := opts.ToServerCreateMap()
			server := createMap["server"].(map[string]interface{})
			serverSecurityGroups := server["security_groups"].([]map[string]interface{})
			serverNetworks := server["networks"].([]map[string]interface{})

			Expect(server["name"]).To(ContainSubstring("vm-"))
			Expect(server["imageRef"]).To(Equal("the_stemcell_id"))
			Expect(serverNetworks[0]["uuid"]).To(Equal("the_net_id"))
			Expect(serverSecurityGroups[0]["name"]).To(Equal("group_1"))
			Expect(serverSecurityGroups[1]["name"]).To(Equal("group_2"))
			Expect(server["availability_zone"]).To(Equal("the_availability_zone"))
			Expect(server["flavorRef"]).To(Equal("the_flavor_id"))
			Expect(opts.(keypairs.CreateOptsExt).KeyName).To(Equal("the_key_name"))
		})

		It("creates a server", func() {
			computeService.CreateServer(
				apiv1.StemcellCID{},
				properties.CreateVM{InstanceType: "the_instance_type"},
				&networkConfig,
				config.OpenstackConfig{StateTimeOut: 10},
			)

			Expect(computeFacade.CreateServerCallCount()).To(Equal(1))
		})

		It("returns an error if the server creation fails", func() {
			computeFacade.CreateServerReturns(nil, errors.New("boom"))

			serverID, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				properties.CreateVM{InstanceType: "the_instance_type"},
				&networkConfig,
				config.OpenstackConfig{StateTimeOut: 10},
			)

			Expect(err.Error()).To(Equal("failed to create server: boom"))
			Expect(serverID).To(Equal(""))
		})

		It("waits for the server to become ACTIVE", func() {
			computeFacade.GetServerReturnsOnCall(0, &servers.Server{ID: "123-456", Status: "not-active"}, nil)
			computeFacade.GetServerReturnsOnCall(1, &servers.Server{ID: "123-456", Status: "ACTIVE"}, nil)

			services.ComputeServicePollingInterval = 0

			serverID, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				properties.CreateVM{InstanceType: "the_instance_type"},
				&networkConfig,
				config.OpenstackConfig{StateTimeOut: 10},
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(serverID).To(Equal("123-456"))
			Expect(computeFacade.GetServerCallCount()).To(Equal(2))
		})

		It("returns an error while waiting if getting server information fails", func() {
			computeFacade.GetServerReturns(&servers.Server{}, errors.New("boom"))

			serverID, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				properties.CreateVM{InstanceType: "the_instance_type"},
				&networkConfig,
				config.OpenstackConfig{StateTimeOut: 10},
			)

			Expect(err.Error()).To(Equal("failed while waiting on the server creation: failed to retrieve server information: boom"))
			Expect(serverID).To(Equal(""))
		})

		It("returns an error while waiting if the server creation finishes in state ERROR", func() {
			computeFacade.GetServerReturns(&servers.Server{ID: "123-456", Status: "ERROR"}, nil)

			serverID, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				properties.CreateVM{InstanceType: "the_instance_type"},
				&networkConfig,
				config.OpenstackConfig{StateTimeOut: 10},
			)

			Expect(err.Error()).To(Equal("failed while waiting on the server creation: server became ERROR state while waiting to become ACTIVE"))
			Expect(serverID).To(Equal(""))
		})

		It("returns an error while waiting if the server creation finishes in state DELETED", func() {
			computeFacade.GetServerReturns(&servers.Server{ID: "123-456", Status: "DELETED"}, nil)

			serverID, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				properties.CreateVM{InstanceType: "the_instance_type"},
				&networkConfig,
				config.OpenstackConfig{StateTimeOut: 10},
			)

			Expect(err.Error()).To(Equal("failed while waiting on the server creation: server became DELETED state while waiting to become ACTIVE"))
			Expect(serverID).To(Equal(""))
		})

		It("returns an error while waiting if the server creation times out", func() {
			computeFacade.GetServerReturns(&servers.Server{ID: "123-456", Status: "not-active"}, nil)

			serverID, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				properties.CreateVM{InstanceType: "the_instance_type"},
				&networkConfig,
				config.OpenstackConfig{StateTimeOut: 0},
			)

			Expect(err.Error()).To(Equal("failed while waiting on the server creation: timeout while waiting for server to become active"))
			Expect(serverID).To(Equal(""))
		})

		It("returns the id of the created server", func() {
			serverID, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				properties.CreateVM{InstanceType: "the_instance_type"},
				&networkConfig,
				config.OpenstackConfig{StateTimeOut: 10},
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(serverID).To(Equal("123-456"))
		})
	})
})
