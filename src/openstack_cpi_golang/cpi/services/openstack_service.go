package services

import (
	"fmt"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services/facades"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	"github.com/gophercloud/gophercloud"
)

type OpenstackService interface {
	ImageServiceV2(config config.OpenstackConfig) (*gophercloud.ServiceClient, error)
}

type openstackService struct {
	openstackFacade facades.OpenstackFacade
	envVar          utils.EnvVar
}

func NewOpenstackService(openstackFacade facades.OpenstackFacade, envVar utils.EnvVar) OpenstackService {
	return openstackService{
		openstackFacade: openstackFacade,
		envVar:          envVar,
	}
}

func (c openstackService) ImageServiceV2(config config.OpenstackConfig) (*gophercloud.ServiceClient, error) {
	authenticatedClient, err := c.authenticate(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create image service, authentication failed: %w", err)
	}

	endpointOpts := gophercloud.EndpointOpts{
		Region: c.envVar.Get("OS_REGION_NAME"),
	}
	return c.openstackFacade.NewImageServiceV2(authenticatedClient, endpointOpts)
}

func (c openstackService) authenticate(config config.OpenstackConfig) (*gophercloud.ProviderClient, error) {
	opts := gophercloud.AuthOptions{
		IdentityEndpoint: config.AuthURL,
		Username:         config.Username,
		Password:         config.APIKey,
		DomainName:       config.DomainName,
		TenantName:       config.ProjectName,
	}

	return c.openstackFacade.AuthenticatedClient(opts)
}
