package compute

import (
	"fmt"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
)

//counterfeiter:generate . FlavorResolver
type FlavorResolver interface {
	ResolveFlavorForInstanceType(flavorName string) (flavors.Flavor, error)
}

type flavorResolver struct {
	serviceClients utils.ServiceClients
	computeFacade  ComputeFacade
}

func NewFlavorResolver(
	serviceClients utils.ServiceClients,
	computeFacade ComputeFacade,
) flavorResolver {
	return flavorResolver{
		serviceClients: serviceClients,
		computeFacade:  computeFacade,
	}
}

func (f flavorResolver) ResolveFlavorForInstanceType(instanceType string) (flavors.Flavor, error) {
	flavorPages, err := f.computeFacade.ListFlavors(f.serviceClients.RetryableServiceClient, flavors.ListOpts{})
	if err != nil {
		return flavors.Flavor{}, fmt.Errorf("failed to list flavors: %w", err)
	}

	allFlavors, err := f.computeFacade.ExtractFlavors(flavorPages)
	if err != nil {
		return flavors.Flavor{}, fmt.Errorf("failed to extract flavors: %w", err)
	}

	var flavor *flavors.Flavor
	for _, singleFlavor := range allFlavors {
		if singleFlavor.Name == instanceType {
			flavor = &singleFlavor
			break
		}
	}

	if flavor == nil {
		return flavors.Flavor{}, fmt.Errorf("flavor for instance type '%s' not found", instanceType)
	}

	if flavor.Ephemeral > 0 {
		// Ephemeral disk size should be at least the double of the vm total memory size, as agent will need:
		// - vm total memory size for swapon,
		// - the rest for /var/vcap/data
		minEphemeralSize := (flavor.RAM / 1024) * 2
		if flavor.Ephemeral < minEphemeralSize {
			return flavors.Flavor{}, fmt.Errorf("flavor '%s' should have at least %dGb of ephemeral disk", flavor.Name, minEphemeralSize)
		}
	}

	return *flavor, nil
}
