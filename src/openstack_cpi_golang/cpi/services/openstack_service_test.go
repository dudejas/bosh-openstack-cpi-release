package services

import (
	"errors"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services/facades/facadesfakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils/utilsfakes"
	"github.com/gophercloud/gophercloud"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("OpenstackService", func() {
	var openstackFacade facadesfakes.FakeOpenstackFacade
	var serviceClient gophercloud.ServiceClient
	var envVar utilsfakes.FakeEnvVar

	Context("ImageServiceV2", func() {
		BeforeEach(func() {
			openstackFacade = facadesfakes.FakeOpenstackFacade{}
			serviceClient = gophercloud.ServiceClient{}
			envVar = utilsfakes.FakeEnvVar{}
		})

		It("returns a ImageServiceV2 instance", func() {
			openstackFacade.AuthenticatedClientReturns(&gophercloud.ProviderClient{}, nil)
			openstackFacade.NewImageServiceV2Returns(&serviceClient, nil)
			envVar.GetReturns("the_os_region_name")

			client, err := NewOpenstackService(&openstackFacade, &envVar).ImageServiceV2(config.OpenstackConfig{})

			Expect(err).ToNot(HaveOccurred())
			Expect(client).To(Equal(&serviceClient))
		})

		It("authenticates using the cpi config", func() {
			openstackFacade.AuthenticatedClientReturns(&gophercloud.ProviderClient{}, nil)
			openstackFacade.NewImageServiceV2Returns(&serviceClient, nil)
			envVar.GetReturns("the_os_region_name")

			openstackConfig := config.OpenstackConfig{
				AuthURL:     "the_auth_url",
				Username:    "the_username",
				APIKey:      "the_api_key",
				DomainName:  "the_domain_name",
				ProjectName: "the_tenant",
			}

			NewOpenstackService(&openstackFacade, &envVar).ImageServiceV2(openstackConfig)

			opts := openstackFacade.AuthenticatedClientArgsForCall(0)
			Expect(opts).To(Equal(gophercloud.AuthOptions{
				IdentityEndpoint: "the_auth_url",
				Username:         "the_username",
				Password:         "the_api_key",
				DomainName:       "the_domain_name",
				TenantName:       "the_tenant",
			}))
		})

		It("gets the region of the service from the environment", func() {
			authenticatedClient := &gophercloud.ProviderClient{}
			openstackFacade.AuthenticatedClientReturns(authenticatedClient, nil)
			openstackFacade.NewImageServiceV2Returns(&serviceClient, nil)
			envVar.GetReturns("the_os_region_name")

			NewOpenstackService(&openstackFacade, &envVar).ImageServiceV2(config.OpenstackConfig{})

			authenticatedClient, endpointOpts := openstackFacade.NewImageServiceV2ArgsForCall(0)
			Expect(endpointOpts).To(Equal(gophercloud.EndpointOpts{
				Region: "the_os_region_name",
			}))
		})

		It("returns an error on failing authentication", func() {
			openstackFacade.AuthenticatedClientReturns(nil, errors.New("boom"))

			client, err := NewOpenstackService(&openstackFacade, &envVar).ImageServiceV2(config.OpenstackConfig{})

			Expect(err.Error()).To(Equal("failed to create image service, authentication failed: boom"))
			Expect(client).To(BeNil())
		})

	})
})
