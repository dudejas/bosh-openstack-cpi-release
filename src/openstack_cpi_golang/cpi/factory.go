package cpi

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/methods"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
)

type Factory struct {
	fs              boshsys.FileSystem
	uuidGen         boshuuid.Generator
	openstackConfig config.OpenstackConfig
	logger          boshlog.Logger
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
	fs boshsys.FileSystem,
	uuidGen boshuuid.Generator,
	openstackConfig config.OpenstackConfig,
	logger boshlog.Logger,
) Factory {
	return Factory{fs, uuidGen, openstackConfig, logger}
}

func (cpiFactory Factory) New(ctx apiv1.CallContext) (apiv1.CPI, error) {
	openstackConfig := cpiFactory.openstackConfig
	err := openstackConfig.Validate()
	if err != nil {
		return CPI{}, err
	}

	return CPI{
		methods.NewInfoMethod(),

		methods.NewCreateStemcellMethod(),
		methods.NewDeleteStemcellMethod(),

		methods.NewCreateVMMethod(),
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
