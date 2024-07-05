package methods_test

import (
	"errors"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/compute/computefakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/methods"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/network/networkfakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils/utilsfakes"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeleteVMMethod", func() {

	var computeServiceBuilder computefakes.FakeComputeServiceBuilder
	var networkServiceBuilder networkfakes.FakeNetworkServiceBuilder
	var computeService computefakes.FakeComputeService
	var networkService networkfakes.FakeNetworkService
	var logger utilsfakes.FakeLogger

	Context("DELETEVMV", func() {

		BeforeEach(func() {
			computeServiceBuilder = computefakes.FakeComputeServiceBuilder{}
			networkServiceBuilder = networkfakes.FakeNetworkServiceBuilder{}
			logger = utilsfakes.FakeLogger{}

			computeServiceBuilder.BuildReturns(&computeService, nil)
			networkServiceBuilder.BuildReturns(&networkService, nil)
			computeService.DeleteServerReturns(nil)
			networkService.GetPortsReturns([]ports.Port{{ID: "test"}}, nil)
			networkService.DeletePortsReturns(nil)

		})
		//
		It("creates the compute service", func() {
			methods.NewDeleteVMMethod(
				&networkServiceBuilder,
				&computeServiceBuilder,
				config.CpiConfig{},
				&logger,
			).DeleteVM(
				apiv1.NewVMCID("vm-id"),
			)

			Expect(computeServiceBuilder.BuildCallCount()).To(Equal(1))
		})

		It("returns an error if the compute service cannot be retrieved", func() {
			computeServiceBuilder.BuildReturns(nil, errors.New("boom"))

			err := methods.NewDeleteVMMethod(
				&networkServiceBuilder,
				&computeServiceBuilder,
				config.CpiConfig{},
				&logger,
			).DeleteVM(
				apiv1.NewVMCID("vm-id"),
			)

			Expect(err.Error()).To(Equal("failed to create compute service: boom"))
		})

		It("creates the network service", func() {
			methods.NewDeleteVMMethod(
				&networkServiceBuilder,
				&computeServiceBuilder,
				config.CpiConfig{},
				&logger,
			).DeleteVM(
				apiv1.NewVMCID("vm-id"),
			)

			Expect(networkServiceBuilder.BuildCallCount()).To(Equal(1))
		})

		It("returns an error if the network service cannot be retrieved", func() {
			networkServiceBuilder.BuildReturns(nil, errors.New("boom"))

			err := methods.NewDeleteVMMethod(
				&networkServiceBuilder,
				&computeServiceBuilder,
				config.CpiConfig{},
				&logger,
			).DeleteVM(
				apiv1.NewVMCID("vm-id"),
			)

			Expect(err.Error()).To(Equal("failed to create network service: boom"))
		})

		It("get ports has been called once with the correct parameters", func() {
			err := methods.NewDeleteVMMethod(
				&networkServiceBuilder,
				&computeServiceBuilder,
				config.CpiConfig{},
				&logger,
			).DeleteVM(
				apiv1.NewVMCID("vm-id"),
			)

			serverID, _, _ := networkService.GetPortsArgsForCall(0)
			Expect(serverID).To(Equal("vm-id"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if no ports were found", func() {
			networkService.GetPortsReturns(nil, errors.New("boom"))

			err := methods.NewDeleteVMMethod(
				&networkServiceBuilder,
				&computeServiceBuilder,
				config.CpiConfig{},
				&logger,
			).DeleteVM(
				apiv1.NewVMCID("vm-id"),
			)

			Expect(err.Error()).To(Equal("failed to get ports: boom"))

		})

		It("deletes a server", func() {
			err := methods.NewDeleteVMMethod(
				&networkServiceBuilder,
				&computeServiceBuilder,
				config.CpiConfig{},
				&logger,
			).DeleteVM(
				apiv1.NewVMCID("vm-id"),
			)

			serverID, _ := computeService.DeleteServerArgsForCall(0)
			Expect(serverID).To(Equal("vm-id"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if the server deletion fails", func() {
			computeService.DeleteServerReturns(errors.New("boom"))

			err := methods.NewDeleteVMMethod(
				&networkServiceBuilder,
				&computeServiceBuilder,
				config.CpiConfig{},
				&logger,
			).DeleteVM(
				apiv1.NewVMCID("vm-id"),
			)

			Expect(err.Error()).To(Equal("failed to delete server: boom"))
		})

		It("delete ports has been called once with the correct parameters", func() {
			err := methods.NewDeleteVMMethod(
				&networkServiceBuilder,
				&computeServiceBuilder,
				config.CpiConfig{},
				&logger,
			).DeleteVM(
				apiv1.NewVMCID("vm-id"),
			)

			exp := []ports.Port{{ID: "test"}}
			ports := networkService.DeletePortsArgsForCall(0)
			Expect(ports).To(Equal(exp))
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if no ports were deleted", func() {
			networkService.DeletePortsReturns(errors.New("boom"))

			err := methods.NewDeleteVMMethod(
				&networkServiceBuilder,
				&computeServiceBuilder,
				config.CpiConfig{},
				&logger,
			).DeleteVM(
				apiv1.NewVMCID("vm-id"),
			)

			Expect(err.Error()).To(Equal("failed to delete ports: boom"))

		})

	})
})
