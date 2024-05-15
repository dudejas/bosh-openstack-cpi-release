package vm

import (
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services/facades"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/pagination"
)

//counterfeiter:generate . InstanceTypeResolver

// InstanceTypeResolver is responsible for resolving the flavor ID of an instance type.
//
// This functionality is put into a separate interface since it is impossible to mock
// the flavorsDetails that are returned by flavors.ListDetail(client, opts)
type InstanceTypeResolver interface {
	GetInstanceTypeFlavorID(
		flavorName string,
		serviceClient *gophercloud.ServiceClient,
		computeFacade facades.ComputeFacade,
	) (string, error)
}

type instanceTypeResolver struct {
}

func NewInstanceTypeResolver() InstanceTypeResolver {
	return instanceTypeResolver{}
}

func (c instanceTypeResolver) GetInstanceTypeFlavorID(
	flavorName string,
	serviceClient *gophercloud.ServiceClient,
	computeFacade facades.ComputeFacade,
) (string, error) {
	var flavorRef string

	flavorsDetails := computeFacade.ListFlavors(serviceClient, nil)
	flavorsDetails.EachPage(func(page pagination.Page) (bool, error) {
		pageElements, err := flavors.ExtractFlavors(page)
		if err != nil {
			return false, err
		}

		for _, element := range pageElements {
			if element.Name == flavorName {
				flavorRef = element.ID
				return false, nil
			}
		}
		return true, nil
	})

	return flavorRef, nil
}
