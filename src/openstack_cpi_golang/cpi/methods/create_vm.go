package methods

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

type CreateVMMethod struct{}

func NewCreateVMMethod() CreateVMMethod {
	return CreateVMMethod{}
}

func (method CreateVMMethod) CreateVM(
	agentID apiv1.AgentID, stemcellCID apiv1.StemcellCID, cloudProps apiv1.VMCloudProps,
	networks apiv1.Networks, diskCIDs []apiv1.DiskCID, env apiv1.VMEnv) (apiv1.VMCID, error) {

	return apiv1.VMCID{}, nil
}

func (method CreateVMMethod) CreateVMV2(
	agentID apiv1.AgentID, stemcellCID apiv1.StemcellCID, cloudProps apiv1.VMCloudProps,
	networks apiv1.Networks, diskCIDs []apiv1.DiskCID, env apiv1.VMEnv) (apiv1.VMCID, apiv1.Networks, error) {
	return apiv1.VMCID{}, apiv1.Networks{}, nil
}
