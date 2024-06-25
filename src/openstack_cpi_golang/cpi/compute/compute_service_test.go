package compute_test

import (
	"errors"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/compute"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/compute/computefakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/loadbalancer/loadbalancerfakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils/utilsfakes"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/bootfromvolume"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ComputeService", func() {
	var serviceClient gophercloud.ServiceClient
	var computeFacade computefakes.FakeComputeFacade
	var flavorResolver computefakes.FakeFlavorResolver
	var volumeConfigurator computefakes.FakeVolumeConfigurator
	var availabilityZoneProvider computefakes.FakeAvailabilityZoneProvider
	var logger utilsfakes.FakeLogger
	var computeService compute.ComputeService
	var networkConfig properties.NetworkConfig
	var defaultCloudConfig properties.CreateVM
	var loadbalancerServiceBuilder loadbalancerfakes.FakeLoadbalancerServiceBuilder
	var loadbalancerService loadbalancerfakes.FakeLoadbalancerService

	BeforeEach(func() {
		providerClient := gophercloud.ProviderClient{TokenID: "the_token"}
		serviceClient = gophercloud.ServiceClient{ProviderClient: &providerClient}
		computeFacade = computefakes.FakeComputeFacade{}
		flavorResolver = computefakes.FakeFlavorResolver{}
		volumeConfigurator = computefakes.FakeVolumeConfigurator{}
		availabilityZoneProvider = computefakes.FakeAvailabilityZoneProvider{}
		loadbalancerServiceBuilder = loadbalancerfakes.FakeLoadbalancerServiceBuilder{}
		loadbalancerService = loadbalancerfakes.FakeLoadbalancerService{}
		logger = utilsfakes.FakeLogger{}

		loadbalancerServiceBuilder.BuildReturns(&loadbalancerService, nil)

		computeService = compute.NewComputeService(&serviceClient, &computeFacade, &flavorResolver, &volumeConfigurator, &availabilityZoneProvider, &loadbalancerServiceBuilder, &logger)
		compute.ComputeServicePollingInterval = 0
		networkConfig = properties.NetworkConfig{}
		computeFacade.CreateServerReturns(&servers.Server{ID: "123-456"}, nil)
		flavorResolver.ResolveFlavorForInstanceTypeReturns(flavors.Flavor{ID: "the_flavor_id", Name: "the_instance_type", RAM: 4096, Ephemeral: 10}, nil)
		computeFacade.GetOSKeyPairReturns(&keypairs.KeyPair{Name: "the_os_keypair_name"}, nil)
		defaultCloudConfig = properties.CreateVM{InstanceType: "the_instance_type", RootDisk: properties.Disk{Size: 1}}
		availabilityZoneProvider.GetAvailabilityZonesReturns([]string{"z1"})
	})

	Context("CreateServer", func() {
		BeforeEach(func() {
			computeFacade.GetServerReturns(&servers.Server{ID: "123-456", Status: "ACTIVE"}, nil)
		})

		It("resolves flavors by instance type", func() {
			_, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				defaultCloudConfig,
				networkConfig,
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(flavorResolver.ResolveFlavorForInstanceTypeArgsForCall(0)).To(Equal("the_instance_type"))
		})

		It("return error if flavors resolution fails", func() {
			flavorResolver.ResolveFlavorForInstanceTypeReturns(flavors.Flavor{}, errors.New("boom"))

			_, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				defaultCloudConfig,
				networkConfig,
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			Expect(err.Error()).To(ContainSubstring("failed to resolve flavor of instance type 'the_instance_type': boom"))
		})

		It("resolves the key pair via cloud config name", func() {
			computeFacade.GetOSKeyPairReturns(&keypairs.KeyPair{Name: "the_key_name"}, nil)

			computeService.CreateServer(
				apiv1.StemcellCID{},
				properties.CreateVM{
					InstanceType: "the_instance_type",
					KeyName:      "key_name_from_properties",
					RootDisk:     properties.Disk{Size: 0},
				},
				networkConfig,
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			_, keyPairName, _ := computeFacade.GetOSKeyPairArgsForCall(0)
			Expect(keyPairName).To(Equal("key_name_from_properties"))
		})

		It("resolves the key pair via openstack config name", func() {
			computeFacade.GetOSKeyPairReturns(&keypairs.KeyPair{Name: "the_key_name"}, nil)

			computeService.CreateServer(
				apiv1.StemcellCID{},
				defaultCloudConfig,
				networkConfig,
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "key_name_from_config"},
			)

			_, keyPairName, _ := computeFacade.GetOSKeyPairArgsForCall(0)
			Expect(keyPairName).To(Equal("key_name_from_config"))
		})

		It("returns an error if key pair name IS NOT PROVIDED", func() {
			computeFacade.GetOSKeyPairReturns(nil, errors.New("boom"))

			_, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				defaultCloudConfig,
				networkConfig,
				config.OpenstackConfig{StateTimeOut: 10},
			)

			Expect(err.Error()).To(Equal("failed to resolve keypair: key pair name undefined"))
		})

		It("returns an error id key pair name cannot be resolved", func() {
			computeFacade.GetOSKeyPairReturns(nil, errors.New("boom"))

			_, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				defaultCloudConfig,
				networkConfig,
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			Expect(err.Error()).To(Equal("failed to resolve keypair: failed to retrieve 'the_key_name': boom"))
		})

		It("returns an error if the disksize is 0 in flavor and cloud properties", func() {
			volumeConfigurator.ConfigureVolumesReturns(nil, errors.New("boom"))

			_, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				properties.CreateVM{InstanceType: "the_instance_type", RootDisk: properties.Disk{Size: 0}},
				networkConfig,
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "key_name_from_config"},
			)

			Expect(err.Error()).To(ContainSubstring("failed to configure volumes: boom"))
		})

		It("creates ops for the server", func() {
			volumeConfigurator.ConfigureVolumesReturns([]bootfromvolume.BlockDevice{{
				UUID:                "the-stemcell-id",
				SourceType:          bootfromvolume.SourceImage,
				DestinationType:     bootfromvolume.DestinationVolume,
				VolumeSize:          999,
				BootIndex:           0,
				DeleteOnTermination: true,
			}}, nil)

			networkConfig = properties.NetworkConfig{
				ManualNetworks: []properties.Network{
					{IP: "1.2.3.4", CloudProps: properties.CreateVMNetwork{NetID: "the_net_id"}},
				},
				SecurityGroups: []string{"group_1", "group_2"},
			}

			bootfromvolume := true

			_, err := computeService.CreateServer(
				apiv1.NewStemcellCID("the_stemcell_id"),
				properties.CreateVM{
					InstanceType:     "the_instance_type",
					AvailabilityZone: "z1",
					RootDisk:         properties.Disk{Size: 1},
					BootFromVolume:   &bootfromvolume,
				},
				networkConfig,
				config.OpenstackConfig{DefaultKeyName: "the_key_name"},
			)
			Expect(err).ToNot(HaveOccurred())

			sClient, opts := computeFacade.CreateServerArgsForCall(0)
			Expect(sClient).To(Equal(&serviceClient))

			createMap, err := opts.ToServerCreateMap()
			server := createMap["server"].(map[string]interface{})
			serverSecurityGroups := server["security_groups"].([]map[string]interface{})
			serverNetworks := server["networks"].([]map[string]interface{})
			blockDevice := server["block_device_mapping_v2"].([]map[string]interface{})

			Expect(server["name"]).To(ContainSubstring("vm-"))
			Expect(server["imageRef"]).To(Equal("the_stemcell_id"))
			Expect(serverNetworks[0]["uuid"]).To(Equal("the_net_id"))
			Expect(serverSecurityGroups[0]["name"]).To(Equal("group_1"))
			Expect(serverSecurityGroups[1]["name"]).To(Equal("group_2"))
			Expect(server["availability_zone"]).To(Equal("z1"))
			Expect(server["flavorRef"]).To(Equal("the_flavor_id"))
			Expect(server["key_name"]).To(Equal("the_os_keypair_name"))
			Expect(blockDevice[0]["uuid"]).To(Equal("the-stemcell-id"))
			Expect(blockDevice[0]["volume_size"]).To(Equal(999.0))
		})

		It("runs server creation in multiple AZs on creation failure", func() {
			availabilityZoneProvider.GetAvailabilityZonesReturns([]string{"z1", "z2"})

			computeFacade.CreateServerReturnsOnCall(0, nil, errors.New("boom"))
			computeFacade.CreateServerReturnsOnCall(1, &servers.Server{ID: "123-456"}, nil)

			computeService.CreateServer(
				apiv1.StemcellCID{},
				defaultCloudConfig,
				networkConfig,
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			_, opts := computeFacade.CreateServerArgsForCall(0)
			createMap, _ := opts.ToServerCreateMap()
			server := createMap["server"].(map[string]interface{})
			Expect(server["availability_zone"]).To(Equal("z1"))

			_, opts = computeFacade.CreateServerArgsForCall(1)
			createMap, _ = opts.ToServerCreateMap()
			server = createMap["server"].(map[string]interface{})
			Expect(server["availability_zone"]).To(Equal("z2"))

			Expect(computeFacade.CreateServerCallCount()).To(Equal(2))
		})

		It("runs server creation in multiple AZs if waiting in server fails", func() {
			availabilityZoneProvider.GetAvailabilityZonesReturns([]string{"z1", "z2"})

			computeFacade.GetServerReturns(&servers.Server{ID: "123-456", Status: "not-active"}, nil)

			compute.ComputeServicePollingInterval = 0

			computeService.CreateServer(
				apiv1.StemcellCID{},
				defaultCloudConfig,
				networkConfig,
				config.OpenstackConfig{StateTimeOut: 0, DefaultKeyName: "the_key_name"},
			)

			_, opts := computeFacade.CreateServerArgsForCall(0)
			createMap, _ := opts.ToServerCreateMap()
			server := createMap["server"].(map[string]interface{})
			Expect(server["availability_zone"]).To(Equal("z1"))

			_, opts = computeFacade.CreateServerArgsForCall(1)
			createMap, _ = opts.ToServerCreateMap()
			server = createMap["server"].(map[string]interface{})
			Expect(server["availability_zone"]).To(Equal("z2"))

			Expect(computeFacade.CreateServerCallCount()).To(Equal(2))
		})

		It("returns an error if the server creation fails", func() {
			computeFacade.CreateServerReturns(nil, errors.New("boom"))

			server, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				defaultCloudConfig,
				networkConfig,
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			Expect(err.Error()).To(Equal("failed to create server in availability zone 'z1': boom"))
			Expect(server).To(BeNil())
		})

		It("waits for the server to become ACTIVE", func() {
			computeFacade.GetServerReturnsOnCall(0, &servers.Server{ID: "123-456", Status: "not-active"}, nil)
			computeFacade.GetServerReturnsOnCall(1, &servers.Server{ID: "123-456", Status: "ACTIVE"}, nil)

			server, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				defaultCloudConfig,
				networkConfig,
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(server.ID).To(Equal("123-456"))
			Expect(computeFacade.GetServerCallCount()).To(Equal(2))
		})

		It("returns an error while waiting if getting server information fails", func() {
			computeFacade.GetServerReturns(nil, errors.New("boom"))

			server, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				defaultCloudConfig,
				networkConfig,
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			Expect(err.Error()).To(Equal("failed while waiting on the server creation in availability zone 'z1': failed to retrieve server information: boom"))
			Expect(server).To(BeNil())
		})

		It("returns an error while waiting if the server creation finishes in state ERROR", func() {
			computeFacade.GetServerReturns(&servers.Server{ID: "123-456", Status: "ERROR"}, nil)

			server, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				defaultCloudConfig,
				networkConfig,
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			Expect(err.Error()).To(Equal("failed while waiting on the server creation in availability zone 'z1': server became ERROR state while waiting to become ACTIVE"))
			Expect(server).To(BeNil())
		})

		It("returns an error while waiting if the server creation finishes in state DELETED", func() {
			computeFacade.GetServerReturns(&servers.Server{ID: "123-456", Status: "DELETED"}, nil)

			server, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				defaultCloudConfig,
				networkConfig,
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			Expect(err.Error()).To(Equal("failed while waiting on the server creation in availability zone 'z1': server became DELETED state while waiting to become ACTIVE"))
			Expect(server).To(BeNil())
		})

		It("returns an error while waiting if the server creation times out", func() {
			computeFacade.GetServerReturns(&servers.Server{ID: "123-456", Status: "not-active"}, nil)

			server, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				defaultCloudConfig,
				networkConfig,
				config.OpenstackConfig{StateTimeOut: 0, DefaultKeyName: "the_key_name"},
			)

			Expect(err.Error()).To(Equal("failed while waiting on the server creation in availability zone 'z1': timeout while waiting for server to become active"))
			Expect(server).To(BeNil())
		})

		It("returns the id of the created server", func() {
			server, err := computeService.CreateServer(
				apiv1.StemcellCID{},
				defaultCloudConfig,
				networkConfig,
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(server.ID).To(Equal("123-456"))
		})
	})

	Context("DeleteServer", func() {
		BeforeEach(func() {
			computeFacade.GetServerReturnsOnCall(0, &servers.Server{ID: "123-456", Status: "ACTIVE"}, nil)
			computeFacade.GetServerReturnsOnCall(1, &servers.Server{ID: "123-456", Status: "DELETED"}, nil)
			computeFacade.GetServerTagsReturns([]string{"tag1", "lbaas_pool_poolID/memberID"}, nil)
			loadbalancerService.DeletePoolMemberReturns(nil)
			computeFacade.DeleteServerReturns(nil)
		})

		It("deletes a server without raising errors", func() {
			err := computeService.DeleteServer(
				"123-456",
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(computeFacade.GetServerCallCount()).To(Equal(2))
			Expect(computeFacade.DeleteServerCallCount()).To(Equal(1))
			Expect(serviceClient.RetryFunc).ToNot(Equal(nil))
		})

		It("returns an error if getServer fails", func() {
			computeFacade.GetServerReturnsOnCall(0, nil, errors.New("boom"))

			err := computeService.DeleteServer(
				"123-456",
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to retrieve server information: boom"))
		})

		It("still succeeds if no server is found", func() {
			computeFacade.GetServerReturnsOnCall(0, nil, errors.New("Resource not found"))

			err := computeService.DeleteServer(
				"123-456",
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			Expect(err).ToNot(HaveOccurred())
		})

		It("deletes a pool member for tags with prefix 'lbaas_pool_'", func() {
			err := computeService.DeleteServer(
				"123-456",
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			poolID, memberID := loadbalancerService.DeletePoolMemberArgsForCall(0)

			Expect(poolID).To(Equal("poolID"))
			Expect(memberID).To(Equal("memberID"))
			Expect(err).ToNot(HaveOccurred())
			Expect(computeFacade.GetServerTagsCallCount()).To(Equal(1))
			Expect(loadbalancerService.DeletePoolMemberCallCount()).To(Equal(1))
		})

		It("does not remove pool memberships if no server tags are found", func() {
			computeFacade.GetServerTagsReturns(nil, errors.New("Resource not found"))

			err := computeService.DeleteServer(
				"123-456",
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(loadbalancerService.DeletePoolMemberCallCount()).To(Equal(0))
			Expect(computeFacade.DeleteServerCallCount()).To(Equal(1))
		})

		It("returns an error if tags retrieval fail", func() {
			computeFacade.GetServerTagsReturns(nil, errors.New("boom"))

			err := computeService.DeleteServer(
				"123-456",
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to retrieve server tags: boom"))
		})

		It("returns an error if building loadbalancerService fails", func() {
			computeFacade.GetServerTagsReturns([]string{"serverTag"}, nil)
			loadbalancerServiceBuilder.BuildReturns(nil, errors.New("boom"))

			err := computeService.DeleteServer(
				"123-456",
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to create loadbalancer service: boom"))
		})

		It("returns an error if deleting a pool member fails", func() {
			loadbalancerService.DeletePoolMemberReturns(errors.New("boom"))

			err := computeService.DeleteServer(
				"123-456",
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to delete pool member: boom"))
		})

		It("returns an error if delete server fails", func() {
			computeFacade.DeleteServerReturns(errors.New("boom"))
			err := computeService.DeleteServer(
				"123-456",
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to delete server: boom"))
		})

		It("waits for the server to become TERMINATED", func() {
			computeFacade.GetServerReturnsOnCall(1, &servers.Server{ID: "123-456", Status: "TERMINATED"}, nil)

			err := computeService.DeleteServer(
				"123-456",
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(computeFacade.GetServerCallCount()).To(Equal(2))
		})

		It("still succeeds if server is not found while deletion", func() {
			computeFacade.GetServerReturnsOnCall(1, nil, errors.New("Resource not found"))

			err := computeService.DeleteServer(
				"123-456",
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(computeFacade.DeleteServerCallCount()).To(Equal(1))
		})

		It("raises an error if it times out while waiting for the server to become DELETED", func() {
			computeFacade.GetServerReturnsOnCall(1, &servers.Server{ID: "123-456", Status: "ACTIVE"}, nil)
			computeFacade.GetServerReturns(&servers.Server{ID: "123-456", Status: "ACTIVE"}, nil)

			err := computeService.DeleteServer(
				"123-456",
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed while waiting on the server deletion: timeout while waiting for server to become deleted"))
		})

		It("raises an error if server status has changed to ERROR instead of DELETED", func() {
			computeFacade.GetServerReturnsOnCall(1, &servers.Server{ID: "123-456", Status: "ERROR"}, nil)

			err := computeService.DeleteServer(
				"123-456",
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed while waiting on the server deletion: server became ERROR state while waiting to become DELETED"))
		})

		It("raises an error if it server retrieval fails while waiting for the server to become DELETED", func() {
			computeFacade.GetServerReturnsOnCall(1, nil, errors.New("boom"))

			err := computeService.DeleteServer(
				"123-456",
				config.OpenstackConfig{StateTimeOut: 10, DefaultKeyName: "the_key_name"},
			)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed while waiting on the server deletion: failed to retrieve server information: boom"))
		})
	})

	Context("SetMetadata", func() {
		var server servers.Server

		BeforeEach(func() {
			server = servers.Server{ID: "123-456"}
		})

		It("does not set tags if none are provided", func() {
			serverTags := properties.ServerTags{}

			err := computeService.SetMetadata(server, serverTags)

			Expect(err).ToNot(HaveOccurred())
			Expect(computeFacade.SetServerMetadataCallCount()).To(Equal(0))
		})

		It("does set tags", func() {
			server := servers.Server{ID: "123-456"}
			serverTags := properties.ServerTags{
				"tag1": "value1",
				"tag2": "value2",
			}

			computeService.SetMetadata(server, serverTags)

			_, serverID, metadata := computeFacade.SetServerMetadataArgsForCall(0)

			Expect(serverID).To(Equal("123-456"))
			Expect(metadata).To(Equal(servers.MetadatumOpts{"tag1": "value1", "tag2": "value2"}))
		})

		It("returns an error if setting metadata fails", func() {
			computeFacade.SetServerMetadataReturns(nil, errors.New("boom"))
			serverTags := properties.ServerTags{
				"tag1": "value1",
				"tag2": "value2",
			}

			err := computeService.SetMetadata(server, serverTags)

			Expect(err.Error()).To(Equal("failed to set metadata: boom"))
		})
	})
})
