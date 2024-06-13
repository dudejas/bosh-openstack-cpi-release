package compute_test

import (
	"errors"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/compute"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/compute/computefakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/mocks"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FlavorResolver", func() {

	var serviceClient gophercloud.ServiceClient
	var computeFacade computefakes.FakeComputeFacade
	var flavorsPage mocks.MockPage

	BeforeEach(func() {
		providerClient := gophercloud.ProviderClient{TokenID: "the_token"}
		serviceClient = gophercloud.ServiceClient{ProviderClient: &providerClient}
		computeFacade = computefakes.FakeComputeFacade{}

		computeFacade.ListFlavorsReturns(flavorsPage, nil)
		computeFacade.ExtractFlavorsReturns([]flavors.Flavor{{ID: "the_flavor_id", Name: "the_instance_type", RAM: 4096, Ephemeral: 10}}, nil)
	})

	Context("CreateServer", func() {
		It("lists flavors", func() {
			compute.NewFlavorResolver(&serviceClient, &computeFacade).ResolveFlavorForInstanceType("the_instance_type")

			Expect(computeFacade.ListFlavorsCallCount()).To(Equal(1))
		})

		It("return error if list flavors fails", func() {
			computeFacade.ListFlavorsReturns(nil, errors.New("boom"))

			_, err := compute.NewFlavorResolver(&serviceClient, &computeFacade).ResolveFlavorForInstanceType("the_instance_type")

			Expect(err.Error()).To(ContainSubstring("failed to list flavors: boom"))
		})

		It("extract flavors", func() {
			compute.NewFlavorResolver(&serviceClient, &computeFacade).ResolveFlavorForInstanceType("the_instance_type")

			Expect(computeFacade.ExtractFlavorsArgsForCall(0)).To(Equal(flavorsPage))
			Expect(computeFacade.ExtractFlavorsCallCount()).To(Equal(1))
		})

		It("return error if extract flavors fails", func() {
			computeFacade.ExtractFlavorsReturns(nil, errors.New("boom"))

			_, err := compute.NewFlavorResolver(&serviceClient, &computeFacade).ResolveFlavorForInstanceType("the_instance_type")

			Expect(err.Error()).To(ContainSubstring("failed to extract flavors: boom"))
		})

		It("return an error if flavor name is not found", func() {
			_, err := compute.NewFlavorResolver(&serviceClient, &computeFacade).ResolveFlavorForInstanceType("not_existing_instance_type")

			Expect(err.Error()).To(ContainSubstring("flavor for instance type 'not_existing_instance_type' not found"))
		})

		It("return an error if flavor ephemeral disk is to small", func() {
			computeFacade.ExtractFlavorsReturns([]flavors.Flavor{{ID: "the_flavor_id", Name: "the_instance_type", RAM: 4096, Ephemeral: 2}}, nil)

			_, err := compute.NewFlavorResolver(&serviceClient, &computeFacade).ResolveFlavorForInstanceType("the_instance_type")

			Expect(err.Error()).To(ContainSubstring("flavor 'the_instance_type' should have at least 8Gb of ephemeral disk"))
		})
	})
})
