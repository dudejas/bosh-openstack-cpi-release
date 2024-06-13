package methods_test

import (
	"errors"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/compute/computefakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/image/imagefakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/methods"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/network/networkfakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils/utilsfakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateVMMethod", func() {

	var computeServiceBuilder computefakes.FakeComputeServiceBuilder
	var networkServiceBuilder networkfakes.FakeNetworkServiceBuilder
	var imageServiceBuilder imagefakes.FakeImageServiceBuilder
	var computeService computefakes.FakeComputeService
	var networkService networkfakes.FakeNetworkService
	var imageService imagefakes.FakeImageService
	var logger utilsfakes.FakeLogger
	var props map[string]interface{}
	var networks apiv1.Networks

	Context("CreateVMV2", func() {

		BeforeEach(func() {
			computeServiceBuilder = computefakes.FakeComputeServiceBuilder{}
			networkServiceBuilder = networkfakes.FakeNetworkServiceBuilder{}
			imageServiceBuilder = imagefakes.FakeImageServiceBuilder{}
			computeService = computefakes.FakeComputeService{}
			networkService = networkfakes.FakeNetworkService{}
			imageService = imagefakes.FakeImageService{}
			logger = utilsfakes.FakeLogger{}

			computeServiceBuilder.BuildReturns(&computeService, nil)
			networkServiceBuilder.BuildReturns(&networkService, nil)
			imageServiceBuilder.BuildReturns(&imageService, nil)
			computeService.CreateServerReturns("123-456", nil)
			networkService.ConfigureNetworkReturns(nil)

			props = map[string]interface{}{
				"instance_type": "the_instance_type",
			}

			networks = apiv1.Networks{
				"network1": apiv1.NewNetwork(apiv1.NetworkOpts{}),
			}
		})
		//
		It("creates the compute service", func() {
			methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				config.OpenstackConfig{},
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.NewVMCloudPropsFromMap(props),
				networks,
				[]apiv1.DiskCID{},
				apiv1.VMEnv{},
			)

			Expect(computeServiceBuilder.BuildCallCount()).To(Equal(1))
		})

		It("returns an error if the compute service cannot be retrieved", func() {
			computeServiceBuilder.BuildReturns(nil, errors.New("boom"))

			stemcellCID, networks, err := methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				config.OpenstackConfig{},
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.NewVMCloudPropsFromMap(props),
				networks,
				[]apiv1.DiskCID{},
				apiv1.VMEnv{},
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
				config.OpenstackConfig{},
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.NewVMCloudPropsFromMap(props),
				networks,
				[]apiv1.DiskCID{},
				apiv1.VMEnv{},
			)

			Expect(networkServiceBuilder.BuildCallCount()).To(Equal(1))
		})

		It("returns an error if the network service cannot be retrieved", func() {
			networkServiceBuilder.BuildReturns(nil, errors.New("boom"))

			stemcellCID, networks, err := methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				config.OpenstackConfig{},
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.NewVMCloudPropsFromMap(props),
				networks,
				[]apiv1.DiskCID{},
				apiv1.VMEnv{},
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
				config.OpenstackConfig{},
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.NewVMCloudPropsFromMap(props),
				networks,
				[]apiv1.DiskCID{},
				apiv1.VMEnv{},
			)

			Expect(imageServiceBuilder.BuildCallCount()).To(Equal(1))
		})

		It("returns an error if the image service cannot be retrieved", func() {
			imageServiceBuilder.BuildReturns(nil, errors.New("boom"))

			stemcellCID, networks, err := methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				config.OpenstackConfig{},
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.NewVMCloudPropsFromMap(props),
				networks,
				[]apiv1.DiskCID{},
				apiv1.VMEnv{},
			)

			Expect(err.Error()).To(Equal("failed to create image service: boom"))
			Expect(stemcellCID).To(Equal(apiv1.VMCID{}))
			Expect(networks).To(Equal(apiv1.Networks{}))
		})

		It("returns an error if the stemcell cannot be found", func() {
			imageService.GetImageReturns("", errors.New("boom"))

			stemcellCID, networks, err := methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				config.OpenstackConfig{},
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.NewVMCloudPropsFromMap(props),
				networks,
				[]apiv1.DiskCID{},
				apiv1.VMEnv{},
			)

			Expect(err.Error()).To(ContainSubstring("failed to resolve stemcell: boom"))
			Expect(stemcellCID).To(Equal(apiv1.VMCID{}))
			Expect(networks).To(Equal(apiv1.Networks{}))
		})

		It("returns an error if the network config creation fails", func() {
			networks = apiv1.Networks{
				"network":                          apiv1.NewNetwork(apiv1.NetworkOpts{Type: "dynamic"}),
				"forbidden_second_dynamic_network": apiv1.NewNetwork(apiv1.NetworkOpts{Type: "dynamic"}),
			}

			stemcellCID, networks, err := methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				config.OpenstackConfig{},
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.NewVMCloudPropsFromMap(props),
				networks,
				[]apiv1.DiskCID{},
				apiv1.VMEnv{},
			)

			Expect(err.Error()).To(ContainSubstring("failed to create network config: invalid dynamic network configuration"))
			Expect(stemcellCID).To(Equal(apiv1.VMCID{}))
			Expect(networks).To(Equal(apiv1.Networks{}))
		})

		It("creates a server", func() {
			methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				config.OpenstackConfig{},
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.NewVMCloudPropsFromMap(props),
				networks,
				[]apiv1.DiskCID{},
				apiv1.VMEnv{},
			)

			stemcellCID, _, _, _ := computeService.CreateServerArgsForCall(0)
			Expect(stemcellCID.AsString()).To(Equal("stemcell-id"))
		})

		It("returns an error if the server creation fails", func() {
			computeService.CreateServerReturns("", errors.New("boom"))

			stemcellCID, networks, err := methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				config.OpenstackConfig{},
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.NewVMCloudPropsFromMap(props),
				networks,
				[]apiv1.DiskCID{},
				apiv1.VMEnv{},
			)

			Expect(err.Error()).To(Equal("failed to create server: boom"))
			Expect(stemcellCID).To(Equal(apiv1.VMCID{}))
			Expect(networks).To(Equal(apiv1.Networks{}))
		})

		It("configures the network of the created server", func() {
			methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				config.OpenstackConfig{},
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.NewVMCloudPropsFromMap(props),
				networks,
				[]apiv1.DiskCID{},
				apiv1.VMEnv{},
			)

			serverID, _ := networkService.ConfigureNetworkArgsForCall(0)
			Expect(serverID).To(Equal("123-456"))
		})

		It("returns an error if the network configuration fails", func() {
			networkService.ConfigureNetworkReturns(errors.New("boom"))

			stemcellCID, networks, err := methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				config.OpenstackConfig{},
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.NewVMCloudPropsFromMap(props),
				networks,
				[]apiv1.DiskCID{},
				apiv1.VMEnv{},
			)

			Expect(err.Error()).To(Equal("failed to configure network for server 123-456: boom"))
			Expect(stemcellCID).To(Equal(apiv1.VMCID{}))
			Expect(networks).To(Equal(apiv1.Networks{}))
		})

		It("returns a server ID", func() {
			stemcellCID, _, err := methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				config.OpenstackConfig{},
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.NewVMCloudPropsFromMap(props),
				networks,
				[]apiv1.DiskCID{},
				apiv1.VMEnv{},
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(stemcellCID.AsString()).To(Equal("123-456"))
		})

		It("returns networks", func() {
			_, networks, err := methods.NewCreateVMMethod(
				&imageServiceBuilder,
				&networkServiceBuilder,
				&computeServiceBuilder,
				config.OpenstackConfig{},
				&logger,
			).CreateVMV2(
				apiv1.NewAgentID("the_agent-id"),
				apiv1.NewStemcellCID("stemcell-id"),
				apiv1.NewVMCloudPropsFromMap(props),
				networks,
				[]apiv1.DiskCID{},
				apiv1.VMEnv{},
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(networks["network1"]).To(Equal(apiv1.NewNetwork(apiv1.NetworkOpts{})))
		})
	})
})
