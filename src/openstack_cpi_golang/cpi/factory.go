package cpi

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/compute"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/image"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/image/root_image"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/methods"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/network"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/openstack"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
)

type Factory struct {
	openstackConfig config.OpenstackConfig
	logger          utils.Logger
}

type CPI struct {
	methods.InfoMethod

	methods.CreateStemcellMethod
	methods.DeleteStemcellMethod

	methods.CreateVMMethod
	methods.DeleteVMMethod
	methods.CalculateVMCloudPropertiesMethod
	methods.HasVMMethod
	methods.RebootVMMethod
	methods.SetVMMetadataMethod
	methods.GetDisksMethod

	methods.CreateDiskMethod
	methods.DeleteDiskMethod
	methods.AttachDiskMethod
	methods.DetachDiskMethod
	methods.HasDiskMethod
	methods.ResizeDiskMethod
	methods.SetDiskMetadataMethod

	methods.DeleteSnapshotMethod
	methods.SnapshotDiskMethod
}

func NewFactory(
	openstackConfig config.OpenstackConfig,
	logger utils.Logger,
) Factory {
	return Factory{openstackConfig, logger}
}

func (cpiFactory Factory) New(ctx apiv1.CallContext) (apiv1.CPI, error) {
	openstackService := openstack.NewOpenstackService(openstack.NewOpenstackFacade(), utils.NewEnvVar())
	openstackConfig := cpiFactory.openstackConfig

	return CPI{
		methods.NewInfoMethod(),

		methods.NewCreateStemcellMethod(
			image.NewImageServiceBuilder(openstackService, openstackConfig, cpiFactory.logger),
			image.NewHeavyStemcellCreator(openstackConfig),
			image.NewLightStemcellCreator(openstackConfig),
			root_image.NewRootImage(),
			cpiFactory.openstackConfig,
			cpiFactory.logger,
		),
		methods.NewDeleteStemcellMethod(
			image.NewImageServiceBuilder(openstackService, openstackConfig, cpiFactory.logger),
			cpiFactory.logger,
		),

		methods.NewCreateVMMethod(
			image.NewImageServiceBuilder(openstackService, openstackConfig, cpiFactory.logger),
			network.NewNetworkServiceBuilder(openstackService, openstackConfig, cpiFactory.logger),
			compute.NewComputeServiceBuilder(openstackService, openstackConfig, cpiFactory.logger),
			cpiFactory.openstackConfig,
			cpiFactory.logger,
		),
		methods.NewDeleteVMMethod(),
		methods.NewCalculateVMCloudPropertiesMethod(),
		methods.NewHasVMMethod(),
		methods.NewRebootVMMethod(),
		methods.NewSetVMMetadataMethod(),
		methods.NewGetDisksMethod(),

		methods.NewCreateDiskMethod(),
		methods.NewDeleteDiskMethod(),
		methods.NewAttachDiskMethod(),
		methods.NewDetachDiskMethod(),
		methods.NewHasDiskMethod(),
		methods.NewResizeDiskMethod(),
		methods.NewSetDiskMetadataMethod(),
		methods.NewDeleteSnapshotMethod(),
		methods.NewSnapshotDiskMethod(),
	}, nil
}
