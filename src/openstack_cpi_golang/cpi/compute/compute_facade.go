package compute

import (
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/availabilityzones"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/pagination"
)

type ServerWithAZ struct {
	servers.Server
	availabilityzones.ServerAvailabilityZoneExt
}

//counterfeiter:generate . ComputeFacade
type ComputeFacade interface {
	CreateServer(client utils.ServiceClient, opts servers.CreateOptsBuilder) (*servers.Server, error)

	DeleteServer(client utils.RetryableServiceClient, serverID string) error

	RebootServer(client utils.ServiceClient, serverID string, opts servers.RebootOptsBuilder) error

	GetServer(client utils.RetryableServiceClient, serverID string) (*servers.Server, error)

	GetServerWithAZ(client utils.RetryableServiceClient, serverID string) (*ServerWithAZ, error)

	ListFlavors(client utils.RetryableServiceClient, opts flavors.ListOpts) (pagination.Page, error)

	ExtractFlavors(page pagination.Page) ([]flavors.Flavor, error)

	GetOSKeyPair(client utils.RetryableServiceClient, keyPairName string, ops keypairs.GetOpts) (*keypairs.KeyPair, error)

	GetServerMetadata(client utils.RetryableServiceClient, serverID string) (map[string]string, error)

	SetServerMetadata(client utils.ServiceClient, serverID string, opts servers.MetadatumOpts) (map[string]string, error)

	UpdateServer(client utils.ServiceClient, serverID string, opt servers.UpdateOptsBuilder) (*servers.Server, error)

	UpdateServerMetadata(client utils.ServiceClient, serverID string, opts servers.UpdateMetadataOptsBuilder) (map[string]string, error)

	DeleteServerMetaData(client *gophercloud.ServiceClient, serverID string, key string) error
}

type computeFacade struct {
}

func NewComputeFacade() computeFacade {
	return computeFacade{}
}

func (c computeFacade) CreateServer(client utils.ServiceClient, opts servers.CreateOptsBuilder) (*servers.Server, error) {
	return servers.Create(client, opts).Extract()
}

func (c computeFacade) DeleteServer(client utils.RetryableServiceClient, serverID string) error {
	return servers.Delete(client, serverID).ExtractErr()
}

func (c computeFacade) RebootServer(client utils.ServiceClient, serverID string, opts servers.RebootOptsBuilder) error {
	return servers.Reboot(client, serverID, opts).ExtractErr()
}

func (c computeFacade) GetServer(client utils.RetryableServiceClient, serverID string) (*servers.Server, error) {
	return servers.Get(client, serverID).Extract()
}

func (c computeFacade) GetServerWithAZ(client utils.RetryableServiceClient, serverID string) (*ServerWithAZ, error) {
	var serverWithAz ServerWithAZ
	err := servers.Get(client, serverID).ExtractInto(&serverWithAz)
	return &serverWithAz, err
}

func (c computeFacade) ListFlavors(client utils.RetryableServiceClient, opts flavors.ListOpts) (pagination.Page, error) {
	return flavors.ListDetail(client, opts).AllPages()
}

func (c computeFacade) ExtractFlavors(page pagination.Page) ([]flavors.Flavor, error) {
	return flavors.ExtractFlavors(page)
}

func (c computeFacade) GetOSKeyPair(client utils.RetryableServiceClient, keyPairName string, opts keypairs.GetOpts) (*keypairs.KeyPair, error) {
	return keypairs.Get(client, keyPairName, opts).Extract()
}

func (c computeFacade) GetServerMetadata(client utils.RetryableServiceClient, serverID string) (map[string]string, error) {
	return servers.Metadata(client, serverID).Extract()
}

func (c computeFacade) SetServerMetadata(client utils.ServiceClient, serverID string, opts servers.MetadatumOpts) (map[string]string, error) {
	return servers.CreateMetadatum(client, serverID, opts).Extract()
}

func (c computeFacade) UpdateServer(client utils.ServiceClient, serverID string, opts servers.UpdateOptsBuilder) (*servers.Server, error) {
	return servers.Update(client, serverID, opts).Extract()
}

func (c computeFacade) UpdateServerMetadata(client utils.ServiceClient, serverID string, opts servers.UpdateMetadataOptsBuilder) (map[string]string, error) {
	return servers.UpdateMetadata(client, serverID, opts).Extract()
}

func (c computeFacade) DeleteServerMetaData(client *gophercloud.ServiceClient, serverID string, key string) error {
	return servers.DeleteMetadatum(client, serverID, key).ExtractErr()
}
