package methods_test

import (
	"errors"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/compute/computefakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/image/imagefakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/loadbalancer/loadbalancerfakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/methods"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/network/networkfakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils/utilsfakes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/pools"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateVMMethod", func() {

	var computeServiceBuilder computefakes.FakeComputeServiceBuilder
	var networkServiceBuilder networkfakes.FakeNetworkServiceBuilder
	var imageServiceBuilder imagefakes.FakeImageServiceBuilder
	var loadbalancerServiceBuilder loadbalancerfakes.FakeLoadbalancerServiceBuilder
	var computeService computefakes.FakeComputeService
	var networkService networkfakes.FakeNetworkService
	var imageService imagefakes.FakeImageService
	var loadbalancerService loadbalancerfakes.FakeLoadbalancerService
	var logger utilsfakes.FakeLogger
	var networks apiv1.Networks
	var jsonStr string
	var cpiConfig config.CpiConfig
	var env apiv1.VMEnv
	var networkConfig properties.NetworkConfig
	var port ports.Port

	Context("CreateVMV2", func() {

		BeforeEach(func() {
			computeServiceBuilder = computefakes.FakeComputeServiceBuilder{}
			networkServiceBuilder = networkfakes.FakeNetworkServiceBuilder{}
			imageServiceBuilder = imagefakes.FakeImageServiceBuilder{}
			loadbalancerServiceBuilder = loadbalancerfakes.FakeLoadbalancerServiceBuilder{}
			computeService = computefakes.FakeComputeService{}
			networkService = networkfakes.FakeNetworkService{}
			imageService = imagefakes.FakeImageService{}
			loadbalancerService = loadbalancerfakes.FakeLoadbalancerService{}
			logger = utilsfakes.FakeLogger{}
			env = apiv1.VMEnv{}

			computeServiceBuilder.BuildReturns(&computeService, nil)
			networkServiceBuilder.BuildReturns(&networkService, nil)
			imageServiceBuilder.BuildReturns(&imageService, nil)
			loadbalancerServiceBuilder.BuildReturns(&loadbalancerService, nil)
			computeService.CreateServerReturns(&servers.Server{ID: "123-456"}, nil)
			networkService.ConfigureVIPNetworkReturns(nil)

			cpiConfig = config.CpiConfig{}
			cpiConfig.Cloud.Properties.Openstack = config.OpenstackConfig{IgnoreServerAvailabilityZone: true}

			networkConfig = properties.NetworkConfig{
				DefaultNetwork: properties.Network{
					Type:       "manual",
					IP:         "1.1.1.1",
					CloudProps: properties.NetworkCloudProps{NetID: "the-net-id"},
				},
				ManualNetworks: []properties.Network{{
					Key:        "key-1",
					Type:       "manual",
					IP:         "1.1.1.1",
					CloudProps: properties.NetworkCloudProps{NetID: "the-net-id-1"},
				}, {
					Key:        "key-2",
					Type:       "manual",
					IP:         "2.2.2.2",
					CloudProps: properties.NetworkCloudProps{NetID: "the-net-id-2"},
				}},
			}
			networkService.GetNetworkConfigurationReturns(networkConfig, nil)
			port = ports.Port{ID: "the-port-id"}
			networkService.CreatePortReturns(port, nil)

			networks = apiv1.Networks{}

			jsonStr = `{
					"instance_type": "type1",
					"loadbalancer_pools": [{"name": "the-pool-name","port": 1234,"monitoring_port": 5678}],
					"availability_zones": ["z1", "z2"]
					
				}`
			networkService.GetSubnetIDReturns("the-subnet-id", nil)
			loadbalancerService.GetPoolReturnsOnCall(0, pools.Pool{ID: "the-pool-id"}, nil)
			loadbalancerService.GetPoolReturnsOnCall(1, pools.Pool{ID: "the-pool-id-1"}, nil)
			loadbalancerService.CreatePoolMemberReturns(&pools.Member{ID: "the-member-id", PoolID: "the-pool-id"}, nil)
		})

		It("creates the compute service", func() {
			methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				&loadbalancerServiceBuilder,
				cpiConfig,
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
				networks,
				[]apiv1.DiskCID{},
				env,
			)

			Expect(computeServiceBuilder.BuildCallCount()).To(Equal(1))
		})

		It("returns an error if the compute service cannot be retrieved", func() {
			computeServiceBuilder.BuildReturns(nil, errors.New("boom"))

			stemcellCID, networks, err := methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				&loadbalancerServiceBuilder,
				cpiConfig,
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
				networks,
				[]apiv1.DiskCID{},
				env,
			)

			Expect(err.Error()).To(Equal("failed to create compute service: boom"))
			Expect(stemcellCID).To(Equal(apiv1.VMCID{}))
			Expect(networks).To(Equal(apiv1.Networks{}))
		})

		It("creates the network service", func() {
			methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				&loadbalancerServiceBuilder,
				cpiConfig,
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
				networks,
				[]apiv1.DiskCID{},
				env,
			)

			Expect(networkServiceBuilder.BuildCallCount()).To(Equal(1))
		})

		It("returns an error if the network service cannot be retrieved", func() {
			networkServiceBuilder.BuildReturns(nil, errors.New("boom"))

			stemcellCID, networks, err := methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				&loadbalancerServiceBuilder,
				cpiConfig,
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
				networks,
				[]apiv1.DiskCID{},
				env,
			)

			Expect(err.Error()).To(Equal("failed to create networking service: boom"))
			Expect(stemcellCID).To(Equal(apiv1.VMCID{}))
			Expect(networks).To(Equal(apiv1.Networks{}))
		})

		It("creates the image service", func() {
			methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				&loadbalancerServiceBuilder,
				cpiConfig,
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
				networks,
				[]apiv1.DiskCID{},
				env,
			)

			Expect(imageServiceBuilder.BuildCallCount()).To(Equal(1))
		})

		It("returns an error if the image service cannot be retrieved", func() {
			imageServiceBuilder.BuildReturns(nil, errors.New("boom"))

			stemcellCID, networks, err := methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				&loadbalancerServiceBuilder,
				cpiConfig,
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
				networks,
				[]apiv1.DiskCID{},
				env,
			)

			Expect(err.Error()).To(Equal("failed to create image service: boom"))
			Expect(stemcellCID).To(Equal(apiv1.VMCID{}))
			Expect(networks).To(Equal(apiv1.Networks{}))
		})

		It("creates the loadbalancer service", func() {
			methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				&loadbalancerServiceBuilder,
				cpiConfig,
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
				networks,
				[]apiv1.DiskCID{},
				env,
			)

			Expect(loadbalancerServiceBuilder.BuildCallCount()).To(Equal(1))
		})

		It("returns an error if the loadbalancer service cannot be retrieved", func() {
			loadbalancerServiceBuilder.BuildReturns(nil, errors.New("boom"))

			stemcellCID, networks, err := methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				&loadbalancerServiceBuilder,
				cpiConfig,
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
				networks,
				[]apiv1.DiskCID{},
				env,
			)

			Expect(err.Error()).To(Equal("failed to create loadbalancer service: boom"))
			Expect(stemcellCID).To(Equal(apiv1.VMCID{}))
			Expect(networks).To(Equal(apiv1.Networks{}))
		})

		It("returns an error if the stemcell cannot be found", func() {
			imageService.GetImageReturns("", errors.New("boom"))

			stemcellCID, networks, err := methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				&loadbalancerServiceBuilder,
				cpiConfig,
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
				networks,
				[]apiv1.DiskCID{},
				env,
			)

			Expect(err.Error()).To(ContainSubstring("failed to resolve stemcell: boom"))
			Expect(stemcellCID).To(Equal(apiv1.VMCID{}))
			Expect(networks).To(Equal(apiv1.Networks{}))
		})

		It("returns an error if the network config creation fails", func() {
			networkService.GetNetworkConfigurationReturns(properties.NetworkConfig{}, errors.New("boom"))

			stemcellCID, networks, err := methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				&loadbalancerServiceBuilder,
				cpiConfig,
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
				networks,
				[]apiv1.DiskCID{},
				env,
			)

			Expect(err.Error()).To(ContainSubstring("failed to create network config: boom"))
			Expect(stemcellCID).To(Equal(apiv1.VMCID{}))
			Expect(networks).To(Equal(apiv1.Networks{}))
		})

		It("creates a port per manual network", func() {

			methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				&loadbalancerServiceBuilder,
				cpiConfig,
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
				networks,
				[]apiv1.DiskCID{},
				env,
			)

			Expect(networkService.CreatePortCallCount()).To(Equal(2))
		})

		It("returns an error if port creation fails", func() {
			networkService.CreatePortReturns(ports.Port{}, errors.New("boom"))

			_, _, err := methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				&loadbalancerServiceBuilder,
				cpiConfig,
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
				networks,
				[]apiv1.DiskCID{},
				env,
			)

			Expect(err.Error()).To(Equal("failed to create port: boom"))
		})

		It("configures the created ports in the network config", func() {
			methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				&loadbalancerServiceBuilder,
				cpiConfig,
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
				networks,
				[]apiv1.DiskCID{},
				env,
			)

			manualNetworks := networkConfig.ManualNetworks
			Expect(manualNetworks[0].Port).To(Equal(port))
			Expect(manualNetworks[1].Port).To(Equal(port))
		})

		It("creates a server", func() {
			methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				&loadbalancerServiceBuilder,
				cpiConfig,
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
				networks,
				[]apiv1.DiskCID{},
				env,
			)

			stemcellCID, _, _, agentID, environment, _ := computeService.CreateServerArgsForCall(0)
			Expect(stemcellCID.AsString()).To(Equal("stemcell-id"))
			Expect(agentID.AsString()).To(Equal("the_agent-id"))
			Expect(environment).To(Equal(env))
		})

		It("returns an error if the server creation fails", func() {
			computeService.CreateServerReturns(nil, errors.New("boom"))

			stemcellCID, networks, err := methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				&loadbalancerServiceBuilder,
				cpiConfig,
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
				networks,
				[]apiv1.DiskCID{},
				env,
			)

			Expect(err.Error()).To(Equal("failed to create server: boom"))
			Expect(stemcellCID).To(Equal(apiv1.VMCID{}))
			Expect(networks).To(Equal(apiv1.Networks{}))
		})

		It("configures the VIP network of the created server", func() {
			methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				&loadbalancerServiceBuilder,
				cpiConfig,
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
				networks,
				[]apiv1.DiskCID{},
				env,
			)

			serverID, _ := networkService.ConfigureVIPNetworkArgsForCall(0)
			Expect(serverID).To(Equal("123-456"))
		})

		It("returns an error if the network configuration fails", func() {
			networkService.ConfigureVIPNetworkReturns(errors.New("boom"))

			stemcellCID, networks, err := methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				&loadbalancerServiceBuilder,
				cpiConfig,
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
				networks,
				[]apiv1.DiskCID{},
				env,
			)

			Expect(err.Error()).To(Equal("failed to configure network for server '123-456': boom"))
			Expect(stemcellCID).To(Equal(apiv1.VMCID{}))
			Expect(networks).To(Equal(apiv1.Networks{}))
		})

		Context("Configure loadbalancer pools", func() {
			It("gets pool ids of provided pools", func() {

				methods.NewCreateVMMethod(
					&imageServiceBuilder,
					&networkServiceBuilder,
					&computeServiceBuilder,
					&loadbalancerServiceBuilder,
					cpiConfig,
					&logger,
				).CreateVMV2(
					apiv1.NewAgentID("the_agent-id"),
					apiv1.NewStemcellCID("stemcell-id"),
					apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
					networks,
					[]apiv1.DiskCID{},
					env,
				)

				poolName := loadbalancerService.GetPoolArgsForCall(0)
				Expect(poolName).To(Equal("the-pool-name"))
			})

			It("returns an error if getting pool ids fails", func() {
				loadbalancerService.GetPoolReturnsOnCall(0, pools.Pool{}, errors.New("boom"))

				_, _, err := methods.NewCreateVMMethod(
					&imageServiceBuilder,
					&networkServiceBuilder,
					&computeServiceBuilder,
					&loadbalancerServiceBuilder,
					cpiConfig,
					&logger,
				).CreateVMV2(
					apiv1.NewAgentID("the_agent-id"),
					apiv1.NewStemcellCID("stemcell-id"),
					apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
					networks,
					[]apiv1.DiskCID{},
					env,
				)

				Expect(err.Error()).To(ContainSubstring("failed to get pool ID of pool 'the-pool-name': boom"))
			})

			It("gets subnets of the default network", func() {
				jsonStr := `{
					"instance_type": "type1",
					"loadbalancer_pools": [{"name": "the-pool-name","port": 1234,"monitoring_port": 5678}],
					"availability_zones": ["z1", "z2"]
				}`

				methods.NewCreateVMMethod(
					&imageServiceBuilder,
					&networkServiceBuilder,
					&computeServiceBuilder,
					&loadbalancerServiceBuilder,
					cpiConfig,
					&logger,
				).CreateVMV2(
					apiv1.NewAgentID("the_agent-id"),
					apiv1.NewStemcellCID("stemcell-id"),
					apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
					networks,
					[]apiv1.DiskCID{},
					env,
				)

				Expect(loadbalancerService.CreatePoolMemberCallCount()).To(Equal(1))
			})

			It("returns an error if getting subnets fails", func() {
				networkService.GetSubnetIDReturns("", errors.New("boom"))

				_, _, err := methods.NewCreateVMMethod(
					&imageServiceBuilder,
					&networkServiceBuilder,
					&computeServiceBuilder,
					&loadbalancerServiceBuilder,
					cpiConfig,
					&logger,
				).CreateVMV2(
					apiv1.NewAgentID("the_agent-id"),
					apiv1.NewStemcellCID("stemcell-id"),
					apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
					networks,
					[]apiv1.DiskCID{},
					env,
				)

				Expect(err.Error()).To(ContainSubstring("failed to get subnet: boom"))
			})

			It("Creates a single pool member", func() {
				methods.NewCreateVMMethod(
					&imageServiceBuilder,
					&networkServiceBuilder,
					&computeServiceBuilder,
					&loadbalancerServiceBuilder,
					cpiConfig,
					&logger,
				).CreateVMV2(
					apiv1.NewAgentID("the_agent-id"),
					apiv1.NewStemcellCID("stemcell-id"),
					apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
					networks,
					[]apiv1.DiskCID{},
					env,
				)

				poolID, ip, pool, subnetID, stateTimeOut := loadbalancerService.CreatePoolMemberArgsForCall(0)
				Expect(poolID).To(Equal("the-pool-id"))
				Expect(ip).To(Equal("1.1.1.1"))
				Expect(pool.Name).To(Equal("the-pool-name"))
				Expect(subnetID).To(Equal("the-subnet-id"))
				Expect(stateTimeOut).To(Equal(0))
			})

			It("Creates multiple pool members", func() {
				jsonStr = `{
					"instance_type": "type1",
					"loadbalancer_pools": [
						{"name": "the-pool-name","port": 1234,"monitoring_port": 5678},
						{"name": "the-pool-name-1","port": 1234,"monitoring_port": 5678}
					],
					"availability_zones": ["z1", "z2"]
				}`

				methods.NewCreateVMMethod(
					&imageServiceBuilder,
					&networkServiceBuilder,
					&computeServiceBuilder,
					&loadbalancerServiceBuilder,
					cpiConfig,
					&logger,
				).CreateVMV2(
					apiv1.NewAgentID("the_agent-id"),
					apiv1.NewStemcellCID("stemcell-id"),
					apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
					networks,
					[]apiv1.DiskCID{},
					env,
				)

				poolID, ip, pool, subnetID, stateTimeOut := loadbalancerService.CreatePoolMemberArgsForCall(0)
				Expect(poolID).To(Equal("the-pool-id"))
				Expect(ip).To(Equal("1.1.1.1"))
				Expect(pool.Name).To(Equal("the-pool-name"))
				Expect(subnetID).To(Equal("the-subnet-id"))
				Expect(stateTimeOut).To(Equal(0))

				poolID, ip, pool, subnetID, stateTimeOut = loadbalancerService.CreatePoolMemberArgsForCall(1)
				Expect(poolID).To(Equal("the-pool-id-1"))
				Expect(ip).To(Equal("1.1.1.1"))
				Expect(pool.Name).To(Equal("the-pool-name-1"))
				Expect(subnetID).To(Equal("the-subnet-id"))
				Expect(stateTimeOut).To(Equal(0))
			})

			It("returns an error if pool member creation fails", func() {
				loadbalancerService.CreatePoolMemberReturns(nil, errors.New("boom"))

				_, _, err := methods.NewCreateVMMethod(
					&imageServiceBuilder,
					&networkServiceBuilder,
					&computeServiceBuilder,
					&loadbalancerServiceBuilder,
					cpiConfig,
					&logger,
				).CreateVMV2(
					apiv1.NewAgentID("the_agent-id"),
					apiv1.NewStemcellCID("stemcell-id"),
					apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
					networks,
					[]apiv1.DiskCID{},
					env,
				)

				Expect(err.Error()).To(ContainSubstring("failed to create pool membership of IP '1.1.1.1' in pool 'the-pool-name': boom"))
			})
		})

		It("sets VM metadata", func() {
			jsonStr = `{
					"instance_type": "type1",
					"loadbalancer_pools": [
						{"name": "the-pool-name","port": 1234,"monitoring_port": 5678},
						{"name": "the-pool-name-1","port": 1234,"monitoring_port": 5678}
					],
					"availability_zones": ["z1", "z2"]
				}`

			loadbalancerService.CreatePoolMemberReturnsOnCall(0, &pools.Member{ID: "the-member-id", PoolID: "the-pool-id"}, nil)
			loadbalancerService.CreatePoolMemberReturnsOnCall(1, &pools.Member{ID: "the-member-id-1", PoolID: "the-pool-id-1"}, nil)

			methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				&loadbalancerServiceBuilder,
				cpiConfig,
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
				networks,
				[]apiv1.DiskCID{},
				env,
			)

			_, tags := computeService.SetMetadataArgsForCall(0)
			Expect(tags["lbaas_pool_1"]).To(Equal("the-pool-id/the-member-id"))
			Expect(tags["lbaas_pool_2"]).To(Equal("the-pool-id-1/the-member-id-1"))
		})

		It("returns a server ID and a network spec", func() {
			stemcellCID, networkSpec, err := methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				&loadbalancerServiceBuilder,
				cpiConfig,
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.CloudPropsImpl{RawMessage: []byte(jsonStr)},
				networks,
				[]apiv1.DiskCID{},
				env,
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(stemcellCID.AsString()).To(Equal("123-456"))
			Expect(networkSpec).To(Equal(networks))
		})
	})
})
