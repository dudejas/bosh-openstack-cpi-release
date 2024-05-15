package vm_test

import (
	"encoding/json"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/vm"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

func TestMethods(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vm Suite")
}

func createNetworkConfig(bytes []byte) (vm.NetworkConfig, error) {
	var networks apiv1.Networks
	err := json.Unmarshal(bytes, &networks)
	Expect(err).ToNot(HaveOccurred())

	networkConfig, err := vm.NewNetworkConfig(networks)

	return networkConfig, err
}
